package ui

import (
	"strings"
	"testing"

	"bitbucket.org/digitalisio/gotransfer/internal/fs"
	"github.com/stretchr/testify/assert"
)

func TestPaneMoveCursor(t *testing.T) {
	p := PaneModel{width: 40, height: 10}
	entries := []fs.FileEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	p.SetEntries(entries)
	p.MoveCursor(1)
	assert.Equal(t, 1, p.cursor)
	p.MoveCursor(10) // clamp to last
	assert.Equal(t, 2, p.cursor)
	p.MoveCursor(-10) // clamp to 0
	assert.Equal(t, 0, p.cursor)
}

func TestPaneRenderEmpty(t *testing.T) {
	p := PaneModel{width: 40, height: 5}
	p.SetEntries(nil)
	out := p.Render(true)
	assert.NotEmpty(t, out)
	assert.NotPanics(t, func() { p.Render(false) })
}

func TestPaneSelectedEntry(t *testing.T) {
	p := PaneModel{width: 40, height: 10}
	p.SetEntries([]fs.FileEntry{{Name: "x.txt"}})
	e, ok := p.SelectedEntry()
	assert.True(t, ok)
	assert.Equal(t, "x.txt", e.Name)
}

func TestPaneRenderNarrow(t *testing.T) {
	p := PaneModel{width: 10, height: 5}
	p.SetEntries([]fs.FileEntry{{Name: "very-long-file-name.txt", Size: 1234}})
	assert.NotPanics(t, func() { p.Render(true) })
	out := p.Render(true)
	lines := strings.Split(out, "\n")
	assert.Equal(t, 5, len(lines), "Render must return exactly height lines")

	// Pathologically small — must not panic and must return exact dimensions
	p2 := PaneModel{width: 3, height: 2}
	p2.SetEntries([]fs.FileEntry{{Name: "x"}})
	var out2 string
	assert.NotPanics(t, func() { out2 = p2.Render(true) })
	lines2 := strings.Split(out2, "\n")
	assert.Equal(t, 2, len(lines2), "Render must return exactly height lines for small dimensions")
}

func TestPaneFilter(t *testing.T) {
	p := PaneModel{width: 40, height: 10}
	p.SetEntries([]fs.FileEntry{{Name: "main.go"}, {Name: "README.md"}, {Name: "go.mod"}})
	p.SetFilter("go")
	for _, entry := range p.visible() {
		assert.NotEqual(t, "README.md", entry.Name)
	}
	assert.Len(t, p.visible(), 2)
}

func TestPaneSortBySize(t *testing.T) {
	p := PaneModel{width: 40, height: 10}
	p.SetEntries([]fs.FileEntry{{Name: "big", Size: 1000}, {Name: "small", Size: 1}})
	p.CycleSort() // SortName(0) → SortSize(1), ascending
	assert.Equal(t, "small", p.visible()[0].Name)
}

func TestPaneMultiSelect(t *testing.T) {
	p := PaneModel{width: 40, height: 10}
	p.SetEntries([]fs.FileEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	p.ToggleSelect() // select a, cursor→1
	p.ToggleSelect() // select b, cursor→2
	assert.Len(t, p.SelectedEntries(), 2)
	p.DeselectAll()
	assert.Len(t, p.SelectedEntries(), 1) // fallback to cursor (c at idx 2)
}
