package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func sftpClient(t *testing.T) *sftp.Client {
	t.Helper()
	host := os.Getenv("TEST_SSH_HOST")
	user := os.Getenv("TEST_SSH_USER")
	pass := os.Getenv("TEST_SSH_PASS")
	if host == "" {
		t.Skip("TEST_SSH_HOST not set")
	}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}
	conn, err := ssh.Dial("tcp", host+":22", config)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	client, err := sftp.NewClient(conn)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })
	return client
}

func TestRemoteBrowserList(t *testing.T) {
	b := NewRemoteBrowser(sftpClient(t))
	entries, err := b.List(context.Background(), "/tmp")
	require.NoError(t, err)
	assert.NotNil(t, entries)
}

func TestRemoteBrowserListNonexistent(t *testing.T) {
	b := NewRemoteBrowser(sftpClient(t))
	_, err := b.List(context.Background(), "/nonexistent-path-gotransfer")
	assert.Error(t, err)
}

func TestRemoteBrowserUploadDownload(t *testing.T) {
	rb := NewRemoteBrowser(sftpClient(t)).(RemoteUploader)
	local := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(local, []byte("gotransfer test"), 0644))
	remote := "/tmp/gotransfer-test-upload.txt"
	require.NoError(t, rb.Upload(context.Background(), local, remote, nil))
	dst := filepath.Join(t.TempDir(), "downloaded.txt")
	require.NoError(t, rb.Download(context.Background(), remote, dst, nil))
	got, _ := os.ReadFile(dst)
	assert.Equal(t, "gotransfer test", string(got))
}
