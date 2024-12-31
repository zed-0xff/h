package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// how many bytes can fit in the screen
func screenCapacity() int64 {
	return cols * int64(maxLinesPerPage)
}

func printAt(x, y int, msg string) {
	for i, c := range msg {
		screen.SetCell(x+i, y, tcell.StyleDefault, c)
	}
}

func printAtEx(x, y int, msg string, styleFunc func(int) tcell.Style) {
	for i, c := range msg {
		screen.SetCell(x+i, y, styleFunc(i), c)
	}
}

func ask(prompt, curValue, allowedChars string, termKeys ...tcell.Key) (string, int) {
	w, _ := screen.Size()
	firstKey := true
	buffer := bytes.NewBufferString(curValue)
	cursorPos := buffer.Len()
	printAt(0, maxLinesPerPage, prompt)
	for {
		printAtEx(len(prompt), maxLinesPerPage, fmt.Sprintf("%*s%s", -w, buffer.String()), func(i int) tcell.Style {
			return tcell.StyleDefault.Underline(i == cursorPos)
		})
		screen.Show()

		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if len(termKeys) > 0 {
				for i, key := range termKeys {
					if ev.Key() == key {
						return buffer.String(), i + 1
					}
				}
			}

			switch ev.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return "", -1
			case tcell.KeyEnter:
				return buffer.String(), 0
			case tcell.KeyLeft:
				if cursorPos > 0 {
					cursorPos--
				} else {
					beep()
				}
			case tcell.KeyRight:
				if cursorPos < buffer.Len() {
					cursorPos++
				} else {
					beep()
				}
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if cursorPos > 0 {
					// del char before cursor
					buffer = bytes.NewBuffer(append(buffer.Bytes()[:cursorPos-1], buffer.Bytes()[cursorPos:]...))
					cursorPos--
				} else {
					beep()
				}
			case tcell.KeyRune:
				c := ev.Rune()
				if len(allowedChars) == 0 || strings.Contains(allowedChars, string(c)) {
					if firstKey && c != ' ' {
						buffer.Reset()
						cursorPos = 0
					}
					buffer = bytes.NewBuffer(append(buffer.Bytes()[:cursorPos], append([]byte{byte(c)}, buffer.Bytes()[cursorPos:]...)...))
					cursorPos++
				} else {
					beep()
				}
			}
			firstKey = false
		}
	}
}

func beep() {
	fmt.Print("\a")
}

func askString(prompt, curValue string) string {
	str, _ := ask(prompt, curValue, "")
	return str
}

func askBinStr(prompt, curValue string) string {
	str, _ := ask(prompt, curValue, "0123456789abcdefABCDEF ")
	return str
}

func askInt(prompt string, curValue int64) int64 {
	str, _ := ask(prompt, fmt.Sprintf("%d", curValue), "0123456789abcdefxABCDEFX")
	if str == "" {
		return curValue
	}
	n, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
		beep()
		return curValue
	}
	return n
}

func askHexInt(prompt string, curValue int64) int64 {
	str, _ := ask(prompt, fmt.Sprintf("%x", curValue), "0123456789abcdefxABCDEFX")
	if str == "" {
		return curValue
	}
	n, err := strconv.ParseInt(str, 16, 64)
	if err != nil {
		beep()
		return curValue
	}
	return n
}

func checkInterrupt() bool {
	if screen.HasPendingEvent() {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return true
			case tcell.KeyRune:
				if ev.Rune() == 'q' || ev.Rune() == 'Q' {
					return true
				}
			}
		}
	}
	return false
}

var g_searchMode = 0

func askSearchPattern(pattern []byte) []byte {
	var key int
	var str string

	for {
		if g_searchMode == 0 {
			pattern_str := strings.TrimSpace(toHex(pattern, int64(scrWidth/3), 1))
			str, key = ask("find hex: ", pattern_str, "0123456789abcdefABCDEF ", tcell.KeyTab)
			if key == 0 {
				return fromHex(str)
			}
		} else {
			pattern_str := string(pattern)
			str, key = ask("find text: ", pattern_str, "", tcell.KeyTab)
			if key == 0 {
				return []byte(str)
			}
		}
		switch key {
		case -1:
			// cancel search
			return nil
		case 0:
			// enter
		}
		g_searchMode = 1 - g_searchMode
	}
}
