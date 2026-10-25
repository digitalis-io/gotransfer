package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"bitbucket.org/digitalisio/gotransfer/internal/fs"
)

func TestModelInit(t *testing.T) {
	lb := fs.NewLocalBrowser()
	m := NewModel(lb, lb, t.TempDir(), t.TempDir())
	m.width, m.height = 120, 40
	out := m.View()
	assert.NotEmpty(t, out.Content)
}
