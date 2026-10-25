package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// overlayCenter composites popup centered over base.
// Screen dimensions are derived from base.
func overlayCenter(base, popup string) string {
	baseLines := strings.Split(base, "\n")
	screenH := len(baseLines)
	screenW := 0
	for _, l := range baseLines {
		if w := ansi.StringWidth(l); w > screenW {
			screenW = w
		}
	}

	popupLines := strings.Split(popup, "\n")
	popH := len(popupLines)
	popW := 0
	for _, l := range popupLines {
		if w := ansi.StringWidth(l); w > popW {
			popW = w
		}
	}

	startY := (screenH - popH) / 2
	startX := (screenW - popW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	out := make([]string, screenH)
	for i, line := range baseLines {
		if i < startY || i >= startY+popH {
			out[i] = line
			continue
		}

		popLine := popupLines[i-startY]

		// Pad base line to screenW so we can splice cleanly.
		lineW := ansi.StringWidth(line)
		if lineW < screenW {
			line += strings.Repeat(" ", screenW-lineW)
		}

		// Left segment: everything before startX.
		left := ansi.Truncate(line, startX, "")

		// Right segment: everything after startX+popW.
		rightStart := startX + popW
		right := ""
		if rightStart < screenW {
			// Cut(s, left, right): drops left visual cells, keeps up to right.
			// We want everything after rightStart, so left=rightStart, right=screenW.
			right = ansi.Cut(line, rightStart, screenW)
		}

		out[i] = left + popLine + right
	}

	return strings.Join(out, "\n")
}
