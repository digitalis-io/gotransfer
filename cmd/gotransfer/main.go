package main

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"

	"bitbucket.org/digitalisio/gotransfer/internal/fs"
	"bitbucket.org/digitalisio/gotransfer/internal/ui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gotransfer: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gotransfer user@host [-p port] [-i keyfile]")
	}

	target := args[0]
	at := strings.LastIndex(target, "@")
	if at < 0 {
		return fmt.Errorf("usage: gotransfer user@host [-p port] [-i keyfile]")
	}
	user := target[:at]
	host := target[at+1:]
	port := 22
	var keyFile string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-p":
			if i+1 >= len(args) {
				return fmt.Errorf("-p requires a port number")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("invalid port: %s", args[i])
			}
			port = p
		case "-i":
			if i+1 >= len(args) {
				return fmt.Errorf("-i requires a key file path")
			}
			i++
			keyFile = args[i]
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	authMethods, err := buildAuth(keyFile)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	khPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	hostKeyCallback, err := buildHostKeyCallback(khPath)
	if err != nil {
		return fmt.Errorf("known_hosts: %w", err)
	}

	cfg := &gossh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := gossh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("sftp: %w", err)
	}
	defer sftpClient.Close()

	localCwd, err := os.Getwd()
	if err != nil {
		localCwd = "/"
	}
	remoteCwd, err := sftpClient.Getwd()
	if err != nil {
		remoteCwd = "/"
	}

	localBrowser := fs.NewLocalBrowser()
	remoteBrowser := fs.NewRemoteBrowser(sftpClient)

	model := ui.NewModel(localBrowser, remoteBrowser, localCwd, remoteCwd)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}

func buildAuth(keyFile string) ([]gossh.AuthMethod, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyFile, err)
		}
		signer, err := gossh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("parse key %s: %w", keyFile, err)
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	}

	// Try ssh-agent, fall back to password prompt.
	var methods []gossh.AuthMethod

	if agentMethod := sshAgentAuth(); agentMethod != nil {
		methods = append(methods, agentMethod)
	}

	methods = append(methods, gossh.PasswordCallback(func() (string, error) {
		fmt.Fprint(os.Stderr, "Password: ")
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return string(pw), nil
	}))

	return methods, nil
}

func buildHostKeyCallback(khPath string) (gossh.HostKeyCallback, error) {
	cb, err := knownhosts.New(khPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", khPath, err)
	}

	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		if cb != nil {
			err := cb(hostname, remote, key)
			if err == nil {
				return nil
			}
			// If the host is known but key mismatch, reject immediately.
			var keyErr *knownhosts.KeyError
			if isKeyError(err, &keyErr) && len(keyErr.Want) > 0 {
				return err
			}
		}
		// Unknown host: prompt user.
		fp := fingerprint(key)
		fmt.Fprintf(os.Stderr, "Unknown host %s\nFingerprint: %s\nAdd to known_hosts? [y/N] ", hostname, fp)
		var answer string
		fmt.Fscanln(os.Stdin, &answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return fmt.Errorf("host key rejected by user")
		}
		if err := appendKnownHost(khPath, hostname, key); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save host key: %v\n", err)
		}
		return nil
	}, nil
}

func sshAgentAuth() gossh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}
	return gossh.PublicKeysCallback(agent.NewClient(conn).Signers)
}

func isKeyError(err error, target **knownhosts.KeyError) bool {
	ke, ok := err.(*knownhosts.KeyError)
	if ok {
		*target = ke
	}
	return ok
}

func fingerprint(key gossh.PublicKey) string {
	h := md5.Sum(key.Marshal()) //nolint:gosec
	parts := make([]string, len(h))
	for i, b := range h {
		parts[i] = hex.EncodeToString([]byte{b})
	}
	return "MD5:" + strings.Join(parts, ":")
}

func appendKnownHost(khPath, hostname string, key gossh.PublicKey) error {
	line := knownhosts.Line([]string{hostname}, key) + "\n"
	f, err := os.OpenFile(khPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.WriteString(line)
	f.Close()
	return err
}
