package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func ask(prompt, curValue, allowedChars string) string {
	printAt(0, maxLinesPerPage, prompt)
	screen.Show()

	buffer := bytes.NewBufferString(curValue)
	for {
		printAt(0, maxLinesPerPage, prompt+buffer.String())
		x := len(prompt) + buffer.Len()
		screen.SetCell(x, maxLinesPerPage, tcell.StyleDefault.Reverse(true), ' ')
		screen.SetCell(x+1, maxLinesPerPage, tcell.StyleDefault, ' ')
		screen.Show()

		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return ""
			case tcell.KeyEnter:
				return buffer.String()
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if buffer.Len() > 0 {
					buffer.Truncate(buffer.Len() - 1)
				}
			case tcell.KeyRune:
				c := ev.Rune()
				if len(allowedChars) == 0 || strings.Contains(allowedChars, string(c)) {
					buffer.WriteRune(c)
				} else {
					beep()
				}
			}
		}
	}
}

func beep() {
	fmt.Print("\a")
}

func askString(prompt, curValue string) string {
	return ask(prompt, curValue, "")
}

func askBinStr(prompt, curValue string) string {
	return ask(prompt, curValue, "0123456789abcdefABCDEF ")
}

func askInt(prompt string, curValue int64) int64 {
	s := ask(prompt, fmt.Sprintf("%d", curValue), "0123456789abcdefxABCDEFX")
	if s == "" {
		return curValue
	}
	n, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		beep()
		return curValue
	}
	return n
}

func askHexInt(prompt string, curValue int64) int64 {
	s := ask(prompt, fmt.Sprintf("%x", curValue), "0123456789abcdefxABCDEFX")
	if s == "" {
		return curValue
	}
	n, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		beep()
		return curValue
	}
	return n
}
