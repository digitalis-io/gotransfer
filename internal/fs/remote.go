package fs

import (
	"context"
	"io"
	"os"
	"sort"

	"github.com/pkg/sftp"
)

// RemoteUploader extends Browser with file transfer operations.
type RemoteUploader interface {
	Browser
	// Upload copies a local file to the remote path.
	Upload(ctx context.Context, localPath, remotePath string) error
	// Download copies a remote file to the local path.
	Download(ctx context.Context, remotePath, localPath string) error
}

type remoteBrowser struct {
	client *sftp.Client
}

// NewRemoteBrowser returns a Browser (also implements RemoteUploader) backed by an SFTP client.
func NewRemoteBrowser(client *sftp.Client) Browser {
	return &remoteBrowser{client: client}
}

// List returns directory entries at path. Directories are listed before files;
// within each group entries are sorted by name.
func (r *remoteBrowser) List(_ context.Context, path string) ([]FileEntry, error) {
	infos, err := r.client.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]FileEntry, 0, len(infos))
	for _, fi := range infos {
		entries = append(entries, FileEntry{
			Name:    fi.Name(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
			Mode:    fi.Mode(),
			IsDir:   fi.IsDir(),
			IsLink:  fi.Mode()&os.ModeSymlink != 0,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// Stat returns metadata for the file or directory at path using Lstat.
func (r *remoteBrowser) Stat(_ context.Context, path string) (FileEntry, error) {
	fi, err := r.client.Lstat(path)
	if err != nil {
		return FileEntry{}, err
	}
	return FileEntry{
		Name:    fi.Name(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		Mode:    fi.Mode(),
		IsDir:   fi.IsDir(),
		IsLink:  fi.Mode()&os.ModeSymlink != 0,
	}, nil
}

// Delete removes the file or directory at path. Directories must be empty; pkg/sftp RemoveDirectory fails on non-empty directories.
func (r *remoteBrowser) Delete(ctx context.Context, path string) error {
	entry, err := r.Stat(ctx, path)
	if err != nil {
		return err
	}
	if entry.IsDir {
		return r.client.RemoveDirectory(path)
	}
	return r.client.Remove(path)
}

// Rename moves or renames src to dst.
func (r *remoteBrowser) Rename(_ context.Context, src, dst string) error {
	return r.client.Rename(src, dst)
}

// Mkdir creates the directory at path, including any necessary parents.
func (r *remoteBrowser) Mkdir(_ context.Context, path string) error {
	return r.client.MkdirAll(path)
}

// Upload copies the local file at localPath to remotePath on the SFTP server.
func (r *remoteBrowser) Upload(_ context.Context, localPath, remotePath string) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := r.client.Create(remotePath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

// Download copies the remote file at remotePath to localPath on the local filesystem.
func (r *remoteBrowser) Download(_ context.Context, remotePath, localPath string) error {
	src, err := r.client.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
