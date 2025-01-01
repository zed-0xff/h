package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

var (
	lastProgressTime time.Time
)

const (
	progressInterval = 50 * time.Millisecond
)

func resetProgress() {
	lastProgressTime = time.Now()
}

func updateProgress(pos int64) {
	if time.Since(lastProgressTime) < progressInterval {
		return
	}
	w, _ := screen.Size()
	fw := float64(w)
	fSize := float64(fileSize)
	printAtEx(0, maxLinesPerPage, fmt.Sprintf("%0*X: %*s", offsetWidth, pos-(pos%0x10000), w-offsetWidth, ""), func(x int) tcell.Style {
		return tcell.StyleDefault.Reverse(float64(pos)/fSize >= float64(x)/fw)
	})
	screen.Show()
	lastProgressTime = time.Now()
}
