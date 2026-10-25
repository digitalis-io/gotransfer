package fs

import "context"

// Browser is the interface implemented by both local and remote (SFTP) filesystem adapters.
type Browser interface {
	// List returns the entries in the directory at path.
	List(ctx context.Context, path string) ([]FileEntry, error)

	// Stat returns metadata for the file or directory at path.
	Stat(ctx context.Context, path string) (FileEntry, error)

	// Delete removes the file or directory at path; behaviour for non-empty directories is implementation-defined.
	Delete(ctx context.Context, path string) error

	// Rename moves or renames src to dst.
	Rename(ctx context.Context, src, dst string) error

	// Mkdir creates the directory at path.
	Mkdir(ctx context.Context, path string) error
}
