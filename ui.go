package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

var (
	stGray = tcell.StyleDefault.Foreground(tcell.NewRGBColor(0x30, 0x30, 0x30))
	stErr  = tcell.StyleDefault.Foreground(tcell.NewRGBColor(0xFF, 0x00, 0x00))

	showBin     bool = false
	showHex     bool = true
	showASCII   bool = true
	showUnicode bool = false

	binMode01            = false
	dispMode             = DispModeDump
	numMode              = NumModeHex
	altColorMode    bool = false
	defaultColsMode int  = 0
	customColsMode  bool = false
	unicodeMode     bool = false

	screen    tcell.Screen
	scrWidth  int
	scrHeight int
	elWidth   int   = 1
	cols      int64 = 0

	pageSize int64 = 0

	g_dedup     = true
	bookmarks   [10]int64
	breadcrumbs []Breadcrumb

	lastErrMsg string
	lastMsg    string
)

// 1: "⠁⠂⠄⠈⠐⠠⡀⢀"
// 2: "⠃⠅⠆⠉⠊⠌⠑⠒⠔⠘⠡⠢⠤⠨⠰⡁⡂⡄⡈⡐⡠⢁⢂⢄⢈⢐⢠⣀"
// 3: "⠇⠋⠍⠎⠓⠕⠖⠙⠚⠜⠣⠥⠦⠩⠪⠬⠱⠲⠴⠸⡃⡅⡆⡉⡊⡌⡑⡒⡔⡘⡡⡢⡤⡨⡰⢃⢅⢆⢉⢊⢌⢑⢒⢔⢘⢡⢢⢤⢨⢰⣁⣂⣄⣈⣐⣠"
// 4: "⠏⠗⠛⠝⠞⠧⠫⠭⠮⠳⠵⠶⠹⠺⠼⡇⡋⡍⡎⡓⡕⡖⡙⡚⡜⡣⡥⡦⡩⡪⡬⡱⡲⡴⡸⢇⢋⢍⢎⢓⢕⢖⢙⢚⢜⢣⢥⢦⢩⢪⢬⢱⢲⢴⢸⣃⣅⣆⣉⣊⣌⣑⣒⣔⣘⣡⣢⣤⣨⣰"
// 5: "⠟⠯⠷⠻⠽⠾⡏⡗⡛⡝⡞⡧⡫⡭⡮⡳⡵⡶⡹⡺⡼⢏⢗⢛⢝⢞⢧⢫⢭⢮⢳⢵⢶⢹⢺⢼⣇⣋⣍⣎⣓⣕⣖⣙⣚⣜⣣⣥⣦⣩⣪⣬⣱⣲⣴⣸"
// 6: "⠿⡟⡯⡷⡻⡽⡾⢟⢯⢷⢻⢽⢾⣏⣗⣛⣝⣞⣧⣫⣭⣮⣳⣵⣶⣹⣺⣼"
// 7: "⡿⢿⣟⣯⣷⣻⣽⣾"
// 8: "⣿"

var ASCII_TBL = []rune(
	/*     0x00 - 0x1f */ " ₁₂₃₄₅₆₇₈₉ₐ·····················" +
		/* 0x20 - 0x3f */ " !\"#$%&'()*+,-./0123456789:;<=>?" +
		/* 0x40 - 0x5f */ "@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_" +
		/* 0x60 - 0x7f */ "`abcdefghijklmnopqrstuvwxyz{|}~⡿" +
		/* 0x80 - 0x9f */ "⢀⢁⢂⢃⢄⢅⢆⢇⢈⢉⢊⢋⢌⢍⢎⢏⢐⢑⢒⢓⢔⢕⢖⢗⢘⢙⢚⢛⢜⢝⢞⢟" + // \u2880-\u289f
		/* 0xa0 - 0xbf */ "⢠⢡⢢⢣⢤⢥⢦⢧⢨⢩⢪⢫⢬⢭⢮⢯⢰⢱⢲⢳⢴⢵⢶⢷⢸⢹⢺⢻⢼⢽⢾⢿" + // \u28a0-\u28bf
		/* 0xc0 - 0xdf */ "⣀⣁⣂⣃⣄⣅⣆⣇⣈⣉⣊⣋⣌⣍⣎⣏⣐⣑⣒⣓⣔⣕⣖⣗⣘⣙⣚⣛⣜⣝⣞⣟" + // \u28c0-\u28df
		/* 0xe0 - 0xff */ "⣠⣡⣢⣣⣤⣥⣦⣧⣨⣩⣪⣫⣬⣭⣮⣯⣰⣱⣲⣳⣴⣵⣶⣷⣸⣹⣺⣻⣼⣽⣾⣿", //  \u28e0-\u28ff
)

func draw() {
	screen.Clear()

	if scrWidth == 0 || scrHeight == 0 {
		scrWidth, scrHeight = screen.Size()
		if scrWidth == 0 || scrHeight == 0 {
			screen.Fini()
			fmt.Println("Error getting screen size", scrWidth, scrHeight)
			os.Exit(1)
		}
	}
	if cols == 0 {
		calcDefaultCols(scrWidth)
	}
	maxLinesPerPage = scrHeight - 1
	nextOffset = fileHexDump(reader, maxLinesPerPage)

	printAtSt(0, maxLinesPerPage, ":", stGray)
	shortname := shortenFName(fname, scrWidth-10)
	printAtSt(scrWidth-utf8.RuneCountInString(shortname), maxLinesPerPage, shortname, stGray)

	if len(lastErrMsg) > 0 {
		printAtSt(0, maxLinesPerPage, lastErrMsg, stErr)
	} else if len(lastMsg) > 0 {
		printAt(0, maxLinesPerPage, lastMsg)
	}

	screen.Show()
}

func calcDefaultCols(scrWidth int) {
	if scrWidth < 1 {
		return
	}

	max_w := 1

	for max_w < scrWidth {
		max_w *= 2
	}
	data := make([]byte, max_w*2)
	for i := 0; i < 0x1000 && max_w > 1; i++ { // prevent infinite loop
		cols = int64(max_w) // XXX drawLine2 ASCII output uses that

		w := drawLine2(-1, data[:max_w], 0, len(data))
		if w <= scrWidth {
			break
		}
		if defaultColsMode == 0 {
			max_w /= 2
		} else {
			max_w -= 1
		}
	}

	if max_w%elWidth != 0 {
		max_w -= max_w % elWidth
	}
	if max_w < 1 {
		max_w = 1
	}

	cols = int64(max_w)
}

func printAtBytes(x, y int, msg []byte) {
	for i, c := range msg {
		if x+i >= scrWidth {
			break
		}
		st := tcell.StyleDefault
		if c < 0x20 {
			st = stGray
		}
		screen.SetCell(x+i, y, st, ASCII_TBL[c])
	}
}

func clearLine(y int) {
	for x := 0; x < scrWidth; x++ {
		screen.SetCell(x, y, tcell.StyleDefault, ' ')
	}
}

func printAt(x, y int, msg string) int {
	i := 0
	for _, c := range msg { // for i, c : = range msg  -  will return byte offset of each rune, but we need its index
		if x+i >= scrWidth {
			break
		}
		screen.SetCell(x+i, y, tcell.StyleDefault, c)
		i++
	}
	return i
}

func printAtSt(x, y int, msg string, st tcell.Style) int {
	i := 0
	for _, c := range msg { // for i, c : = range msg  -  will return byte offset of each rune, but we need its index
		if x+i >= scrWidth {
			break
		}
		screen.SetCell(x+i, y, st, c)
		i++
	}
	return i
}

func printAtEx(x, y int, msg string, styleFunc func(int) tcell.Style) {
	for i, c := range msg {
		if x+i >= scrWidth {
			break
		}
		screen.SetCell(x+i, y, styleFunc(i), c)
	}
}

func ask(prompt, curValue, allowedChars string, firstKey bool, termKeys ...tcell.Key) (string, tcell.Key) {
	w, _ := screen.Size()
	buffer := bytes.NewBufferString(curValue)
	cursorPos := buffer.Len()
	printAt(0, maxLinesPerPage, prompt)
	for {
		printAtEx(len(prompt), maxLinesPerPage, fmt.Sprintf("%*s", -w, buffer.String()), func(i int) tcell.Style {
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
	screen.Beep()
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
	str, _ := ask(prompt, fmt.Sprintf("%d", curValue), EXPR_ALLOWED_CHARS, true)
	if str == "" {
		return curValue
	}
	n, err := parseExpr(str)
	if err != nil {
		beep()
		return curValue
	}
	return n
}

// same as askHexInt(), but also support percents, i.e. "goto 50%", keeping current alignment
func askOffset(prompt string, curValue int64) int64 {
	str, _ := ask(prompt, fmt.Sprintf("%x", curValue), EXPR_ALLOWED_CHARS, true)
	if str == "" {
		return curValue
	}
	if str[len(str)-1] == '%' {
		n, err := parseExprRadix(strings.ToLower(str[:len(str)-1]), 10)
		if err != nil || n < 0 || n > 100 {
			beep()
			return curValue
		}
		newOffset := (n * fileSize) / 100
		if elWidth > 1 {
			newOffset -= newOffset % int64(elWidth) // align to element size
			newOffset += offset % int64(elWidth)    // keep current offset alignment
		}
		return newOffset
	}
	n, err := parseExprRadix(strings.ToLower(str), 16)
	if err != nil {
		beep()
		return curValue
	}
	return n
}

func askHexInt(prompt string, curValue int64) int64 {
	str, _ := ask(prompt, fmt.Sprintf("%x", curValue), EXPR_ALLOWED_CHARS, true)
	if str == "" {
		return curValue
	}
	n, err := parseExprRadix(strings.ToLower(str), 16)
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

func waitForAnyKey() {
	for {
		ev := screen.PollEvent()
		switch ev.(type) {
		case *tcell.EventKey:
			return
		}
	}
}

var g_searchMode = 0

func askPattern(prefix string, pattern0 []byte) []byte {
	var key tcell.Key
	var str string

	firstKey := true
	pattern := pattern0
	for {
		if g_searchMode == 0 {
			pattern_str := strings.TrimSpace(toHex(pattern, int64(scrWidth/3), 1))
			str, key = ask(prefix+"hex : ", pattern_str, "0123456789abcdefABCDEF ", firstKey, tcell.KeyTab, tcell.KeyUp, tcell.KeyDown)
			if key == tcell.KeyEnter {
				pattern = fromHex(str)
				searchHistory.Add(g_searchMode, pattern)
				return pattern
			}
		} else {
			pattern_str := string(pattern)
			str, key = ask(prefix+"text: ", pattern_str, "", firstKey, tcell.KeyTab, tcell.KeyUp, tcell.KeyDown)
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

func askCommand() string {
	var key tcell.Key
	var cmd string

	firstKey := true
	for {
		cmd, key = ask("command: ", cmd, "", firstKey, tcell.KeyUp, tcell.KeyDown)
		if key == tcell.KeyEnter {
			commandHistory.Add(cmd)
			return cmd
		}
		switch key {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			// cancel command
			return ""
		case tcell.KeyUp:
			// prev history
			prevCommand := commandHistory.Prev()
			if prevCommand != "" {
				cmd = prevCommand
				firstKey = false
			} else {
				beep()
			}
		case tcell.KeyDown:
			// next history
			nextCommand := commandHistory.Next()
			if nextCommand != "" {
				cmd = nextCommand
				firstKey = false
			} else {
				if cmd == "" {
					// TODO: keep nonfinished user input?
					beep()
				} else {
					firstKey = false
				}
			}
		default:
			// unexpected key
			return ""
		}
	}
}

func showMsg(msg string) {
	lastMsg = msg
}

func showError(err error) {
	showErrStr(err.Error())
}

func showErrStr(chunks ...interface{}) {
	lastErrMsg = fmt.Sprint(chunks...)
	beep()
}

func setCols(c int64) {
	if c < 0 {
		return
	}

	cols = c
	if cols == 0 {
		defaultColsMode = 1 - defaultColsMode
		calcDefaultCols(scrWidth)
		defaultColsMode = 1 - defaultColsMode
	}
}
