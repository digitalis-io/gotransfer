// Package fs provides filesystem abstractions for local and remote file browsing.
package fs

import (
	"fmt"
	"os"
	"time"
)

// FileEntry represents a single directory entry from either a local or remote filesystem.
type FileEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	IsDir   bool
	IsLink  bool
}

// FormatSize returns n as a human-readable string using binary prefixes (B, KB, MB, GB)
// with one decimal place for KB and above.
func FormatSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// FormatMode returns the 10-character Unix permission string for m (e.g. "drwxr-xr-x").
func FormatMode(m os.FileMode) string {
	return m.String()
}
