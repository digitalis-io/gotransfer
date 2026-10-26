package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverlayCenterNoPanic(t *testing.T) {
	base := strings.Repeat("x", 80) + "\n" + strings.Repeat("x", 80)
	popup := "┌────┐\n│ hi │\n└────┘"
	result := overlayCenter(base, popup)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "hi")
}

func TestOverlayCenterNarrowTerminal(t *testing.T) {
	base := strings.Repeat("x", 20) + "\n" + strings.Repeat("x", 20)
	popup := "┌──────────────────────────────┐\n│ wider than base              │\n└──────────────────────────────┘"
	assert.NotPanics(t, func() { overlayCenter(base, popup) })
}
