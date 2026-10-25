package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalBrowserList(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("hi"), 0644)
	os.Mkdir(filepath.Join(tmp, "sub"), 0755)
	b := NewLocalBrowser()
	entries, err := b.List(context.Background(), tmp)
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	assert.Contains(t, names, "a.txt")
	assert.Contains(t, names, "sub")
}

func TestLocalBrowserListEmpty(t *testing.T) {
	tmp := t.TempDir()
	b := NewLocalBrowser()
	entries, err := b.List(context.Background(), tmp)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLocalBrowserMkdirRenameDelete(t *testing.T) {
	tmp := t.TempDir()
	b := NewLocalBrowser()
	require.NoError(t, b.Mkdir(context.Background(), filepath.Join(tmp, "newdir")))
	require.NoError(t, b.Rename(context.Background(),
		filepath.Join(tmp, "newdir"), filepath.Join(tmp, "renamed")))
	require.NoError(t, b.Delete(context.Background(), filepath.Join(tmp, "renamed")))
	entries, _ := b.List(context.Background(), tmp)
	assert.Empty(t, entries)
}
