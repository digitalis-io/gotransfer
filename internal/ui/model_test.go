package ui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bitbucket.org/digitalisio/gotransfer/internal/fs"
)

func TestModelInit(t *testing.T) {
	lb := fs.NewLocalBrowser()
	m := NewModel(lb, lb, t.TempDir(), t.TempDir())
	m.width, m.height = 120, 40
	out := m.View()
	assert.NotEmpty(t, out.Content)
}

func TestModelDeleteKeyEnterConfirm(t *testing.T) {
	lb := fs.NewLocalBrowser()
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "del.txt"), []byte{}, 0644))
	m := NewModel(lb, lb, tmp, tmp)
	m.width, m.height = 120, 40
	entries, err := lb.List(context.Background(), tmp)
	require.NoError(t, err)
	m.left.SetEntries(entries)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	assert.Equal(t, modeConfirm, m2.(Model).mode)
	assert.Equal(t, confirmDelete, m2.(Model).pending)
}

func TestModelRenameKey(t *testing.T) {
	lb := fs.NewLocalBrowser()
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "file.txt"), []byte{}, 0644))
	m := NewModel(lb, lb, tmp, tmp)
	m.width, m.height = 120, 40
	entries, err := lb.List(context.Background(), tmp)
	require.NoError(t, err)
	m.left.SetEntries(entries)
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	assert.Equal(t, modeRename, m2.(Model).mode)
	assert.Equal(t, "file.txt", m2.(Model).inputBuf)
}

func TestModelMkdirKey(t *testing.T) {
	lb := fs.NewLocalBrowser()
	tmp := t.TempDir()
	m := NewModel(lb, lb, tmp, tmp)
	m.width, m.height = 120, 40
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	assert.Equal(t, modeMkdir, m2.(Model).mode)
	assert.Equal(t, "", m2.(Model).inputBuf)
}

func TestModelDeleteConfirmExecutes(t *testing.T) {
	lb := fs.NewLocalBrowser()
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "gone.txt")
	require.NoError(t, os.WriteFile(fpath, []byte{}, 0644))
	m := NewModel(lb, lb, tmp, tmp)
	m.width, m.height = 120, 40
	entries, err := lb.List(context.Background(), tmp)
	require.NoError(t, err)
	m.left.SetEntries(entries)
	// Press Delete to enter confirm
	m2raw, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	m2 := m2raw.(Model)
	assert.Equal(t, modeConfirm, m2.mode)
	// Confirm with 'y'
	_, cmd := m2.Update(tea.KeyPressMsg{Code: 'y'})
	require.NotNil(t, cmd)
	msg := cmd()
	opMsg, ok := msg.(operationDoneMsg)
	require.True(t, ok)
	assert.NoError(t, opMsg.err)
	_, statErr := os.Stat(fpath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestModelMkdirExecutes(t *testing.T) {
	lb := fs.NewLocalBrowser()
	tmp := t.TempDir()
	m := NewModel(lb, lb, tmp, tmp)
	m.width, m.height = 120, 40
	// 'd' enters mkdir mode
	m2raw, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	m2 := m2raw.(Model)
	// type name
	for _, r := range "newdir" {
		m2raw, _ = m2.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m2 = m2raw.(Model)
	}
	assert.Equal(t, "newdir", m2.inputBuf)
	// Enter to create
	_, cmd := m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	opMsg, ok := msg.(operationDoneMsg)
	require.True(t, ok)
	assert.NoError(t, opMsg.err)
	info, err := os.Stat(filepath.Join(tmp, "newdir"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
