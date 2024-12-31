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

func ask(prompt, curValue, allowedChars string, firstKey bool, termKeys ...tcell.Key) (string, tcell.Key) {
	w, _ := screen.Size()
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
				for _, key := range termKeys {
					if ev.Key() == key {
						return buffer.String(), key
					}
				}
			}

			switch ev.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return "", ev.Key()
			case tcell.KeyEnter:
				return buffer.String(), ev.Key()
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
	str, _ := ask(prompt, curValue, "", true)
	return str
}

func askBinStr(prompt, curValue string) string {
	str, _ := ask(prompt, curValue, "0123456789abcdefABCDEF ", true)
	return str
}

func askInt(prompt string, curValue int64) int64 {
	str, _ := ask(prompt, fmt.Sprintf("%d", curValue), "0123456789abcdefxABCDEFX", true)
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
	str, _ := ask(prompt, fmt.Sprintf("%x", curValue), "0123456789abcdefxABCDEFX", true)
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

func askSearchPattern(pattern0 []byte) []byte {
	var key tcell.Key
	var str string

	firstKey := true
	pattern := pattern0
	for {
		if g_searchMode == 0 {
			pattern_str := strings.TrimSpace(toHex(pattern, int64(scrWidth/3), 1))
			str, key = ask("find hex : ", pattern_str, "0123456789abcdefABCDEF ", firstKey, tcell.KeyTab, tcell.KeyUp, tcell.KeyDown)
			if key == tcell.KeyEnter {
				pattern = fromHex(str)
				searchHistory.Add(g_searchMode, pattern)
				return pattern
			}
		} else {
			pattern_str := string(pattern)
			str, key = ask("find text: ", pattern_str, "", firstKey, tcell.KeyTab, tcell.KeyUp, tcell.KeyDown)
			if key == tcell.KeyEnter {
				pattern = []byte(str)
				searchHistory.Add(g_searchMode, pattern)
				return pattern
			}
		}
		switch key {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			// cancel search
			return nil
		case tcell.KeyTab:
			// switch search mode
			g_searchMode = 1 - g_searchMode
		case tcell.KeyUp:
			// prev history
			prevMode, prevPattern := searchHistory.Prev()
			if prevPattern != nil {
				g_searchMode = prevMode
				pattern = prevPattern
				firstKey = false
			} else {
				beep()
			}
		case tcell.KeyDown:
			// next history
			nextMode, nextPattern := searchHistory.Next()
			if nextPattern != nil {
				g_searchMode = nextMode
				pattern = nextPattern
				firstKey = false
			} else {
				if bytes.Equal(pattern, pattern0) {
					beep()
				} else {
					pattern = pattern0
					firstKey = false
				}
			}
		default:
			// unexpected key
			return nil
		}
	}
}
