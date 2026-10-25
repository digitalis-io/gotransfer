package fs

import (
	"context"
	"os"
	"sort"
)

type localBrowser struct{}

// NewLocalBrowser returns a Browser backed by the local filesystem.
func NewLocalBrowser() Browser {
	return &localBrowser{}
}

// List returns directory entries at path. Directories are listed before files;
// within each group entries are sorted by name. Symlinks are detected via Lstat.
func (l *localBrowser) List(_ context.Context, path string) ([]FileEntry, error) {
	des, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]FileEntry, 0, len(des))
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}
		isLink := false
		if de.Type()&os.ModeSymlink != 0 {
			isLink = true
		} else if !de.Type().IsRegular() && !de.IsDir() {
			// double-check via Lstat for unusual types
			fi, lerr := os.Lstat(path + "/" + de.Name())
			if lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
				isLink = true
			}
		}
		entries = append(entries, FileEntry{
			Name:    de.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   de.IsDir(),
			IsLink:  isLink,
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
func (l *localBrowser) Stat(_ context.Context, path string) (FileEntry, error) {
	fi, err := os.Lstat(path)
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

// Delete removes the file or directory tree at path.
func (l *localBrowser) Delete(_ context.Context, path string) error {
	return os.RemoveAll(path)
}

// Rename moves or renames src to dst.
func (l *localBrowser) Rename(_ context.Context, src, dst string) error {
	return os.Rename(src, dst)
}

// Mkdir creates the directory at path, including any necessary parents.
func (l *localBrowser) Mkdir(_ context.Context, path string) error {
	return os.MkdirAll(path, 0755)
}
