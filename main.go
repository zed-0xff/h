package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

type Range struct {
	start int64
	end   int64
}

type Breadcrumb struct {
	offset int64
	key    termbox.Key
}

type Reader interface {
	ReadAt(p []byte, off int64) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
	Read(p []byte) (n int, err error)
	Fd() uintptr
}

const maxMode = 2

var (
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

	sparseMap map[Range]bool = make(map[Range]bool)
	mapReady  bool           = false

	a = 0
	b = 0
	c = 0x100
	z = 0x31
)

// how many bytes are fit in whole screen
func screenCapacity() int64 {
	return cols * int64(maxLinesPerPage)
}

func draw() {
	termbox.SetCursor(0, 0) // needed when ssh-ing into cygwin
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCursor(-1, -1)

	scrWidth, scrHeight = termbox.Size()
	if scrWidth == 0 || scrHeight == 0 {
		termbox.Close()
		fmt.Println("Error getting screen size", scrWidth, scrHeight)
		os.Exit(1)
	}
	maxLinesPerPage = scrHeight - 1

	if mode == 0 && cols > int64(scrWidth/3) {
		mode++
	}

	nextOffset = fileHexDump(reader, maxLinesPerPage)

	printAt(0, maxLinesPerPage, ":")

	//    colorTable()
	termbox.Flush()
}

func printAt(x, y int, msg string) {
	for i, c := range msg {
		termbox.SetCell(x+i, y, c, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func toHexChar(c byte) byte {
	if c < 10 {
		return '0' + c
	} else {
		return 'a' + c - 10
	}
}

func colorTable() {
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	w, h := termbox.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x += 4 {
			value := x/2*a + y*b + z
			text := fmt.Sprintf("%04x", ((value) & 0xffff))
			for i, c := range text {
				termbox.SetCell(x+i, y, c, termbox.Attribute((value)&0x7fff), termbox.ColorDefault)
			}
		}
	}
}

func drawHex(x, y int, buf []byte) int {
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
			color := termbox.ColorDefault
			if leadingZero {
				if octet == 0 {
					color = termbox.Attribute(0x1fff & (0x09 + z*0x100))
				} else {
					leadingZero = false
				}
			}
			termbox.SetCell(x, y, rune(toHexChar(octet)), color, termbox.ColorDefault)
			x++

			octet = byte & 0x0f
			color = termbox.ColorDefault
			if leadingZero {
				if octet == 0 {
					color = termbox.Attribute(0x1fff & (0x09 + z*0x100))
				} else {
					leadingZero = false
				}
			}
			termbox.SetCell(x, y, rune(toHexChar(octet)), color, termbox.ColorDefault)
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
		termbox.Close()
		fmt.Println("Tried to read", len(buf), "bytes at offset", curLineOffset)
		panic(err)
	}

	chunks := make([][]byte, 2) // Create a slice of 2 elements, each of which will be a byte slice
	for i := range chunks {
		chunks[i] = make([]byte, cols) // Create each chunk as a byte slice of length cols
	}
	c := 0

	chunks[c] = buf[0:cols]

	scrWidth, _ = termbox.Size() // Get the screen width before drawing the lines
	chunkPos = cols
	drawLine(0, chunks[c], curLineOffset)
	curLineOffset += cols
	was_separator := false
	c = 1 - c
	iLine := 1
	var curSkip Range

	for iLine < maxLines {
		if time.Since(t0) > 50*time.Millisecond {
			drawLine(iLine, make([]byte, 0), curLineOffset)
			termbox.Flush()
			t0 = time.Now()
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
				if mapReady {
					for k, _ := range sparseMap {
						if curLineOffset >= k.start && curLineOffset < k.end {
							if curLineOffset%cols == 0 {
								curLineOffset = k.end - cols
							} else {
								curLineOffset = k.end - cols*2 + (curLineOffset % cols)
							}
							chunkPos = int64(nRead) // force read
							break
						}
						if k.start > curLineOffset {
							break
						}
					}
				}
			}
		}
		curLineOffset += cols
		chunkPos += cols

		if chunkPos >= int64(nRead) {
			// Copy the previous chunk, because reading into buf will change its contents, and it will break lines deduplication
			chunks[1-c] = append([]byte(nil), chunks[1-c]...)
			nRead, err = f.ReadAt(buf, curLineOffset)
			if err == io.EOF {
				break
			}
			if err != nil {
				termbox.Close()
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

func handleEvents() {
	for {
		dir := 0
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyArrowLeft:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key})
				dir = -1
				offset -= 1
				invalidateSkips()
			case termbox.KeyArrowRight:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key})
				dir = 1
				offset += 1
				invalidateSkips()
			case termbox.KeyArrowDown:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key})
				dir = 1
				offset += cols
			case termbox.KeyArrowUp:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key})
				dir = -1
				offset -= cols
			case termbox.KeyCtrlG:
				offset = askHexInt("[hex] offset: ", offset)
			case termbox.KeyPgdn, termbox.KeySpace:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, termbox.KeyPgdn})
				dir = 1
				offset = nextOffset
			case termbox.KeyPgup:
				// efficiently handle skipping over deduplicated lines
				if len(breadcrumbs) > 0 && breadcrumbs[len(breadcrumbs)-1].key == termbox.KeyPgdn {
					offset = breadcrumbs[len(breadcrumbs)-1].offset
					breadcrumbs = breadcrumbs[:len(breadcrumbs)-1]
				} else {
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, termbox.KeyPgup})
				}
				dir = -1
				offset -= cols * int64(maxLinesPerPage)
			case termbox.KeyHome:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key})
				offset = 0
			case termbox.KeyEnd:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key})
				offset = maxOffset()
			case termbox.KeyBackspace, termbox.KeyBackspace2:
				if len(breadcrumbs) > 0 {
					offset = breadcrumbs[len(breadcrumbs)-1].offset
					breadcrumbs = breadcrumbs[:len(breadcrumbs)-1]
				}
			case termbox.KeyTab:
				mode += 1
				if mode > maxMode {
					mode = 0
				}
			case termbox.KeyEsc, termbox.KeyCtrlC:
				return

			default:
				switch ev.Ch {
				case '-':
					if cols-int64(elWidth) > 0 {
						cols -= int64(elWidth)
						invalidateSkips()
					}
				case '+', '=':
					cols += int64(elWidth)
					invalidateSkips()
				case '0':
					cols = calcDefaultCols()
				case '1', '2', '4', '8':
					elWidth = int(ev.Ch - '0')
				case '9':
					elWidth = 0x10
				case 'd':
					g_dedup = !g_dedup
				case 'g':
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, termbox.KeyHome})
					offset = 0
				case 'G':
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, termbox.KeyEnd})
					offset = maxOffset()
				case 'n':
					searchNext()
				case 'N':
					searchPrev()
				case 'w':
					cols = askInt("width: ", cols)
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
	width, _ := termbox.Size()
	w = int64(width) / 4 / 8 * 8
	data := make([]byte, 0x100)
	for {
		s := toHexLine(data, 0, w, 1) + toAsciiLine(data, w)
		if len(s) <= width {
			return w
		}
		w -= 8
	}
}

// TODO: cache?
func buildSparseMap() {
	fd := int(reader.Fd())
	for pos := int64(0); pos < fileSize; {
		nextHole := findHole(fd, pos)
		if nextHole == -1 {
			break
		}
		nextData := findData(fd, nextHole)
		if nextData == -1 {
			break
		}
		sparseMap[Range{nextHole, nextData}] = true
		pos = nextData
	}
	mapReady = true
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
		offset, err = strconv.ParseInt(os.Args[2], 16, 64)
		if err != nil {
			fmt.Println("Error parsing offset:", err)
			return
		}
	}

	offsetWidth = len(fmt.Sprintf("%X", fileSize))
	if offsetWidth < 8 {
		offsetWidth = 8
	}

	err = termbox.Init()
	if err != nil {
		fmt.Println("Error initializing termbox:", err)
		return
	}

	defer termbox.Close()

	cols = calcDefaultCols()

	go buildSparseMap()

	draw()
	handleEvents()
}
