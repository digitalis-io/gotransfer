package ui

import (
	"fmt"
	"strings"
	"time"
)

// progressModel holds the state of an in-progress file transfer.
type progressModel struct {
	filename string
	written  int64
	total    int64
	done     bool
	startAt  time.Time
}

// Render returns a two-line progress bar string fitting within width columns.
func (p progressModel) Render(width int) string {
	pct := 0
	if p.total > 0 {
		pct = int(p.written * 100 / p.total)
	}

	speed := ""
	if !p.startAt.IsZero() && p.written > 0 {
		elapsed := time.Since(p.startAt).Seconds()
		if elapsed > 0 {
			speed = fmt.Sprintf(" %.1f KB/s", float64(p.written)/elapsed/1024)
		}
	}

	// Reserve space: " NNN%" (5 chars) + speed + some padding.
	overhead := 5 + len(speed) + 2
	barW := width - overhead
	if barW < 4 {
		barW = 4
	}

	filled := barW * pct / 100
	if filled > barW {
		filled = barW
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)

	return fmt.Sprintf("%s\n%s %d%%%s", p.filename, bar, pct, speed)
}
