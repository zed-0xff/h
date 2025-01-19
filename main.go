package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Range struct {
	start int64
	end   int64
}

type Breadcrumb struct {
	offset int64
	key    tcell.Key
}

type Reader interface {
	ReadAt(p []byte, off int64) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
	Read(p []byte) (n int, err error)
	Fd() uintptr
}

const maxMode = 2

var (
	screen          tcell.Screen
	reader          Reader
	fileSize        int64
	offset          int64
	offsetWidth     int
	elWidth         int = 1
	cols            int64
	mode            int = 0
	maxLinesPerPage int
	nextOffset      int64
	scrWidth        int
	scrHeight       int
	g_dedup         bool = true
	breadcrumbs     []Breadcrumb
	skipMap         map[Range]bool = make(map[Range]bool)
	lastErrMsg      string
	defaultColsMode int = 0

	sparseMap []Range = make([]Range, 0)
	mapReady  bool    = false

	a = 0
	b = 0
	c = 0x100
	z = 0x31
)

func findNextData(pos int64) int64 {
	if !mapReady {
		return -1
	}

	for _, r := range sparseMap {
		if pos >= r.start && pos < r.end {
			return r.end
		}
		if r.start > pos {
			break
		}
	}
	return -1
}

func findPrevData(pos int64) int64 {
	if !mapReady {
		return -1
	}

	for i := len(sparseMap) - 1; i >= 0; i-- {
		r := sparseMap[i]
		if pos > r.start && pos <= r.end {
			return r.start
		}
		if r.end < pos {
			break
		}
	}

	return -1
}

func draw() {
	screen.Clear()

	scrWidth, scrHeight = screen.Size()
	if scrWidth == 0 || scrHeight == 0 {
		screen.Fini()
		fmt.Println("Error getting screen size", scrWidth, scrHeight)
		os.Exit(1)
	}
	maxLinesPerPage = scrHeight - 1

	//	if mode == 0 && cols > int64(scrWidth/3) {
	//		mode++
	//	}

	nextOffset = fileHexDump(reader, maxLinesPerPage)

	printAt(0, maxLinesPerPage, ":")

	//    colorTable()
	screen.Show()
}

func toHexChar(c byte) byte {
	if c < 10 {
		return '0' + c
	} else {
		return 'a' + c - 10
	}
}

//func colorTable() {
//	if a < 0 {
//		a = 0
//	}
//	if b < 0 {
//		b = 0
//	}
//	w, h := screen.Size()
//	for y := 0; y < h; y++ {
//		for x := 0; x < w; x += 4 {
//			value := x/2*a + y*b + z
//			text := fmt.Sprintf("%04x", ((value) & 0xffff))
//			for i, c := range text {
//				termbox.SetCell(x+i, y, c, termbox.Attribute((value)&0x7fff), termbox.ColorDefault)
//			}
//		}
//	}
//}

func drawHex(x, y int, buf []byte) int {
	stGray := tcell.StyleDefault.Foreground(tcell.NewRGBColor(0x30, 0x30, 0x30))

	for j := 0; j < len(buf); j += elWidth {
		if elWidth == 1 && j > 0 && j%(8*elWidth) == 0 { // Add an extra space every 8 groups
			x++
		}

		leadingZero := elWidth > 1 || buf[j] == 0

		for k := elWidth - 1; k >= 0; k-- {
			if j+k >= len(buf) {
				continue
			}
			byte := buf[j+k]

			octet := byte >> 4
			st := tcell.StyleDefault
			if leadingZero {
				if octet == 0 {
					st = stGray
				} else {
					leadingZero = false
				}
			}
			screen.SetCell(x, y, st, rune(toHexChar(octet)))
			x++

			octet = byte & 0x0f
			st = tcell.StyleDefault
			if leadingZero {
				if octet == 0 {
					st = stGray
				} else {
					leadingZero = false
				}
			}
			screen.SetCell(x, y, st, rune(toHexChar(octet)))
			x++
		}
		x++
	}

	return x
}

func drawLine(iLine int, chunk []byte, offset int64) {
	printAt(0, iLine, fmt.Sprintf("%0*X:", offsetWidth, offset))
	x := offsetWidth + 2

	ascii := toAsciiLine(chunk, cols)

	switch mode {
	case 0:
		drawHex(x, iLine, chunk)
		printAt(scrWidth-int(cols), iLine, ascii)
	case 1:
		drawHex(x, iLine, chunk)
	case 2:
		printAt(scrWidth-int(cols), iLine, ascii)
	}
}

func fileHexDump(f io.ReaderAt, maxLines int) int64 {
	var chunkPos int64
	t0 := time.Now()

	bufSize := int(cols) * maxLines
	var buf = make([]byte, bufSize)

	curLineOffset := offset
	nRead, err := f.ReadAt(buf, curLineOffset)
	if err != nil && err != io.EOF {
		// stop termbox
		screen.Fini()
		fmt.Println("Tried to read", len(buf), "bytes at offset", curLineOffset)
		panic(err)
	}

	chunks := make([][]byte, 2) // Create a slice of 2 elements, each of which will be a byte slice
	for i := range chunks {
		chunks[i] = make([]byte, cols) // Create each chunk as a byte slice of length cols
	}
	c := 0

	chunks[c] = buf[0:cols]

	scrWidth, _ = screen.Size() // Get the screen width before drawing the lines
	chunkPos = cols
	drawLine(0, chunks[c], curLineOffset)
	curLineOffset += cols
	was_separator := false
	c = 1 - c
	iLine := 1
	var curSkip Range

	for iLine < maxLines {
		if time.Since(t0) > progressInterval {
			drawLine(iLine, make([]byte, 0), curLineOffset)
			screen.Show()
			t0 = time.Now()
			if checkInterrupt() {
				return curLineOffset
			}
		}

		chunks[c] = buf[chunkPos : chunkPos+cols]
		if !g_dedup || !bytes.Equal(chunks[c], chunks[1-c]) {
			if was_separator {
				was_separator = false
				curSkip.end = curLineOffset
				skipMap[curSkip] = true
				curSkip = Range{}
			}

			drawLine(iLine, chunks[c], curLineOffset)
			iLine++
			c = 1 - c
		} else {
			// cur line equals to previous and dedup is on
			if !was_separator {
				curSkip.start = curLineOffset
				was_separator = true
				printAt(0, iLine, "*")
				iLine++
			} else {
				// separator already drawn
				nextData := findNextData(curLineOffset)
				if nextData != -1 {
					if curLineOffset%cols == 0 {
						curLineOffset = nextData - cols
					} else {
						curLineOffset = nextData - cols*2 + (curLineOffset % cols)
					}
					chunkPos = int64(nRead) // force read
				}
			}
		}
		curLineOffset += cols
		chunkPos += cols

		if chunkPos >= int64(nRead) {
			// Copy the previous chunk, because reading into buf will change its contents, and it will break lines deduplication
			chunks[1-c] = append([]byte(nil), chunks[1-c]...)
			nRead, err = f.ReadAt(buf, curLineOffset)
			if nRead == 0 {
				break
			}
			if err != nil && err != io.EOF {
				screen.Fini()
				panic(err)
			}
			chunkPos = 0
		}
	}

	// Draw the last line if it was a separator or EOF
	if iLine < maxLines && (was_separator || nRead == 0) {
		drawLine(iLine, make([]byte, 0), fileSize)
	}
	return curLineOffset
}

func toHex(buf []byte, cols int64, width int) string {
	if width != 1 && width != 2 && width != 4 && width != 8 && width != 16 {
		panic(fmt.Sprintf("Invalid width: %d", width))
	}

	var hexBytes string
	for j := 0; j < int(cols); j += width {
		if width == 1 && j > 0 && j%(8*width) == 0 { // Add an extra space every 8 groups
			hexBytes += " "
		}
		if j+width <= len(buf) {
			// Group bytes according to the specified width
			for k := width - 1; k >= 0; k-- {
				hexBytes += fmt.Sprintf("%02x", buf[j+k])
			}
			hexBytes += " "
		} else {
			// Handle incomplete groups or padding
			for k := 0; k < width; k++ {
				if j+k < len(buf) {
					hexBytes += fmt.Sprintf("%02x", buf[j+k])
				} else {
					hexBytes += "  " // Padding
				}
			}
			hexBytes += " "
		}
	}

	return hexBytes
}

func toHexLine(buf []byte, ea int64, cols int64, width int) string {
	return fmt.Sprintf("%0*X: %s", offsetWidth, ea, toHex(buf, cols, width))
}

func toAsciiLine(buf []byte, cols int64) string {
	var asciiRep string
	for j := 0; j < int(cols); j++ {
		var c byte = 0
		if j < len(buf) {
			c = buf[j]
		}

		if c >= 32 && c <= 126 {
			asciiRep += fmt.Sprintf("%c", c)
		} else if c == 0 {
			asciiRep += " "
		} else {
			asciiRep += "."
		}
	}

	return asciiRep
}

func invalidateSkips() {
	skipMap = make(map[Range]bool)
}

func maxOffset() int64 {
	add := offset % cols
	return max(0, fileSize-fileSize%cols-int64(maxLinesPerPage-1)*cols+add)
}

func writeFile(fname string, offset int64, size int64) error {
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = reader.Seek(offset, 0)
	if err != nil {
		return err
	}

	buf := make([]byte, 8*1024*1024)
	for size > 0 {
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}
		if int64(n) > size {
			n = int(size)
		}
		_, err = file.Write(buf[:n])
		if err != nil {
			return err
		}
		size -= int64(n)
	}

	return nil

}

func handleEvents() {
	for {
		dir := 0
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyLeft:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				dir = -1
				offset -= 1
				invalidateSkips()
			case tcell.KeyRight:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				dir = 1
				offset += 1
				invalidateSkips()
			case tcell.KeyDown:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				dir = 1
				offset += cols
			case tcell.KeyUp:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				dir = -1
				offset -= cols
			case tcell.KeyCtrlG:
				offset = askHexInt("[hex] offset: ", offset)
			case tcell.KeyPgDn:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyPgDn})
				dir = 1
				offset = nextOffset
			case tcell.KeyPgUp:
				// efficiently handle skipping over deduplicated lines
				if len(breadcrumbs) > 0 && breadcrumbs[len(breadcrumbs)-1].key == tcell.KeyPgDn {
					offset = breadcrumbs[len(breadcrumbs)-1].offset
					breadcrumbs = breadcrumbs[:len(breadcrumbs)-1]
				} else {
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyPgUp})
				}
				dir = -1
				offset -= cols * int64(maxLinesPerPage)
			case tcell.KeyHome:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				offset = 0
			case tcell.KeyEnd:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				offset = maxOffset()
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(breadcrumbs) > 0 {
					offset = breadcrumbs[len(breadcrumbs)-1].offset
					breadcrumbs = breadcrumbs[:len(breadcrumbs)-1]
				}
			case tcell.KeyTab:
				mode += 1
				if mode > maxMode {
					mode = 0
				}
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return

			case tcell.KeyRune:
				switch ev.Rune() {
				case ' ':
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyPgDn})
					dir = 1
					offset = nextOffset
				case '-':
					defaultColsMode = 0
					if cols-int64(elWidth) > 0 {
						cols -= int64(elWidth)
						invalidateSkips()
					}
				case '+', '=':
					defaultColsMode = 0
					cols += int64(elWidth)
					invalidateSkips()
				case '0':
					cols = calcDefaultCols()
					defaultColsMode = 1 - defaultColsMode
				case '1', '2', '4', '8':
					elWidth = int(ev.Rune() - '0')
				case '9':
					elWidth = 0x10
				case 'd':
					g_dedup = !g_dedup
				case 'g':
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyHome})
					offset = 0
				case 'G':
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyEnd})
					offset = maxOffset()
				case 'n':
					if !searchNext() {
						beep()
					}
				case 'N':
					if !searchPrev() {
						beep()
					}
				case 'w':
					cols = askInt("width: ", cols)
					if cols == 0 {
						defaultColsMode = 1 - defaultColsMode
						cols = calcDefaultCols()
						defaultColsMode = 1 - defaultColsMode
					}
					if mode == 0 && cols > int64(scrWidth/3) {
						mode++
					}
				case 'W':
					fname := askString("write to: ", fmt.Sprintf("%0*x.bin", offsetWidth, offset))
					if fname != "" {
						size := askHexInt("[hex] size: ", 0x1000)
						if size > 0 {
							err := writeFile(fname, offset, size)
							if err != nil {
								beep()
							}
						}
					}
				case '/':
					searchUI()
				case 'q', 'Q':
					return
				}
			}

			if g_dedup && dir != 0 {
				for k, _ := range skipMap {
					if offset >= k.start && offset < k.end {
						if dir == 1 {
							offset = k.end
						} else {
							offset = k.start - cols
						}
						break
					}
				}
			}
			if offset < 0 {
				offset = 0
			} else if offset >= maxOffset() {
				offset = maxOffset()
			}
			draw()
		}
	}
}

func calcDefaultCols() int64 {
	var w int64
	scrWidth, _ := screen.Size()
	w = 1
	for w < int64(scrWidth) {
		w *= 2
	}
	data := make([]byte, 0x200)
	var s string
	for {
		switch mode {
		case 0:
			s = toHexLine(data, 0, w, elWidth) + toAsciiLine(data, w)
		case 1:
			s = toHexLine(data, 0, w, elWidth)
		case 2:
			s = toAsciiLine(data, w)
		}
		if len(s) <= scrWidth {
			break
		}
		if defaultColsMode == 0 {
			w /= 2
		} else {
			w -= 1
		}
	}
	if elWidth == 1 {
		if w%8 != 0 {
			w -= w % 8
		}
	} else {
		if w%int64(elWidth) != 0 {
			w -= w % int64(elWidth)
		}
	}
	return w
}

func printLastErr() {
	if lastErrMsg != "" {
		fmt.Println(lastErrMsg)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: hexdump <file>")
		return
	}

	fname := os.Args[1]

	file, err := os.Open(fname)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	if strings.HasPrefix(fname, "\\\\.\\PhysicalDrive") {
		fileSize, err = getDriveSize(fname)
		if err != nil {
			panic(err)
		}
		reader = NewAlignedReader(file, fileSize, 512)
	} else {
		reader = file
		fileInfo, err := file.Stat()
		if err != nil {
			panic(err)
		}
		fileSize = fileInfo.Size()
	}

	if len(os.Args) > 2 {
		for i := 0; i < len(os.Args); i++ {
			if os.Args[i] == "--debug" {
				os.Args = append(os.Args[:i], os.Args[i+1:]...)
				buildSparseMap()
				fmt.Println("Sparse map:")
				for i, r := range sparseMap {
					fmt.Printf("%2x: %12x %12x\n", i, r.start, r.end)
				}
				os.Exit(0)
			}
		}

		offset, err = strconv.ParseInt(os.Args[2], 16, 64)
		if err != nil {
			fmt.Println("Error parsing offset:", err)
			return
		}
	}

	go initSearch()

	offsetWidth = len(fmt.Sprintf("%X", fileSize))
	if offsetWidth < 8 {
		offsetWidth = 8
	}

	defer printLastErr()

	screen, err = tcell.NewScreen()
	if err != nil {
		panic(err)
	}
	if err := screen.Init(); err != nil {
		panic(err)
	}
	defer screen.Fini()

	cols = calcDefaultCols()
	defaultColsMode = 1 // next mode

	go buildSparseMap()

	draw()
	handleEvents()
}
