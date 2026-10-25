package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgressRender(t *testing.T) {
	p := progressModel{filename: "deploy.sh", written: 620, total: 1000}
	out := p.Render(40)
	assert.Contains(t, out, "deploy.sh")
	assert.Contains(t, out, "62%")
}

func TestProgressRenderComplete(t *testing.T) {
	p := progressModel{filename: "f", written: 100, total: 100, done: true}
	out := p.Render(40)
	assert.Contains(t, out, "100%")
}

func TestProgressRenderZeroTotal(t *testing.T) {
	p := progressModel{filename: "empty.txt", written: 0, total: 0}
	out := p.Render(40)
	assert.Contains(t, out, "empty.txt")
	assert.Contains(t, out, "0%")
}

func TestProgressRenderWithSpeed(t *testing.T) {
	p := progressModel{
		filename: "big.tar",
		written:  512 * 1024,
		total:    1024 * 1024,
		startAt:  time.Now().Add(-time.Second),
	}
	out := p.Render(60)
	assert.Contains(t, out, "big.tar")
	assert.Contains(t, out, "50%")
	assert.Contains(t, out, "KB/s")
}

func TestProgressRenderNarrow(t *testing.T) {
	p := progressModel{filename: "x", written: 50, total: 100}
	out := p.Render(5)
	assert.Contains(t, out, "50%")
}
