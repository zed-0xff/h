package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type DisplayMode int

const (
	NumModeHex DisplayMode = iota
	NumModeBin01
	NumModeBinX
)

const (
	DispModeDump = iota
	DispModeText
	DispModeMax = DispModeText
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

const MaxMode = 3
const TextMode = 3

var (
	reader          Reader
	fileSize        int64
	base            int64 = 0
	baseMult        int64 = 1
	offset          int64
	offsetWidth     int
	maxLinesPerPage int
	nextOffset      int64
	skipMap         map[Range]bool = make(map[Range]bool)
	lastErrMsg      string
	fname           string

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

func toHexChar(c byte) byte {
	if c < 10 {
		return '0' + c
	} else {
		return 'a' + c - 10
	}
}

func drawBin(x, y int, buf []byte, chars []rune, max_width int) int {
	for j := 0; j < len(buf); j += elWidth {
		if elWidth == 1 && j > 0 && j%(8*elWidth) == 0 { // Add an extra space every 8 groups
			x++
		}

		leadingZero := true
		for k := elWidth - 1; k >= 0; k-- {
			if j+k >= len(buf) {
				continue
			}
			mask := byte(0x80)
			byte := buf[j+k]

			for i := 0; i < 8; i++ {
				st := tcell.StyleDefault
				bit := byte & mask
				rune := chars[0]

				if bit == 0 {
					if leadingZero {
						st = stGray
					}
				} else {
					//leadingZero = false
					rune = chars[1]
				}

				screen.SetCell(x, y, st, rune)
				x++
				mask >>= 1
			}
		}
		x++
		if x >= max_width {
			break
		}
	}
	return x
}

func drawHex(x, y int, buf []byte, max_width int) int {

	for j := 0; j < len(buf); j += elWidth {
		if elWidth == 1 && j > 0 && j%(8*elWidth) == 0 { // Add an extra space every 8 groups
			x++
		}

		leadingZero := elWidth > 1 || buf[j] == 0

		for k := elWidth - 1; k >= 0; k-- {
			if j+k >= len(buf) {
				continue
			}
			st0 := tcell.StyleDefault

			byte := buf[j+k]
			if elWidth == 1 && altColorMode && byte < 0x10 {
				st0 = stGray
			}

			octet := byte >> 4
			st := st0
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
			st = st0
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
		if x >= max_width {
			break
		}
	}
	return x
}

func drawLine(iLine int, chunk []byte, offset int64) int {
	return drawLine2(iLine, chunk, offset, scrWidth)
}

// as in IDA's idc.here()
func here() int64 {
	return offset2ea(offset)
}

func offset2ea(offset int64) int64 {
	return base + offset*baseMult
}

// also used for calculating max width
func drawLine2(iLine int, chunk []byte, offset int64, max_width int) int {
	printAt(0, iLine, fmt.Sprintf("%0*X:", offsetWidth, offset2ea(offset)))
	x := offsetWidth + 2

	if showBin {
		if binMode01 {
			x = drawBin(x, iLine, chunk, []rune{'0', '1'}, max_width) + 1
		} else {
			x = drawBin(x, iLine, chunk, []rune{'_', 'X'}, max_width) + 1
		}

		if x >= max_width {
			return x
		}
	}

	if showHex {
		x = drawHex(x, iLine, chunk, max_width) + 1
		if x >= max_width {
			return x
		}
	}

	if showASCII {
		if cols < int64(max_width) && (showBin || showHex) {
			printAtBytes(max_width-int(cols), iLine, chunk)
		} else {
			printAtBytes(x, iLine, chunk)
		}
		x += len(chunk) + 1
	}
	return x
}

func fileHexDump(f io.ReaderAt, maxLines int) int64 {
	var chunkPos int64
	var bufSize int

	t0 := time.Now()
	scrWidth, _ = screen.Size() // Get the screen width before drawing the lines
	maxTextCols := scrWidth - 2 - offsetWidth

	if dispMode == DispModeText {
		bufSize = maxTextCols * maxLines
	} else {
		bufSize = int(cols) * maxLines
	}
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
	c := 0

	if dispMode == DispModeText {
		for i := range chunks {
			chunks[i] = make([]byte, maxTextCols) // Create each chunk as a byte slice of length maxTextCols
		}
		nlPos := bytes.IndexAny(buf, "\r\n")
		if nlPos == -1 {
			nlPos = maxTextCols
		}
		chunks[c] = buf[0:nlPos]
		chunkPos = int64(nlPos)
	} else {
		for i := range chunks {
			chunks[i] = make([]byte, cols) // Create each chunk as a byte slice of length cols
		}
		chunks[c] = buf[0:cols]
		chunkPos = cols
	}

	drawLine(0, chunks[c], curLineOffset)
	curLineOffset += int64(len(chunks[c]))
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

		if dispMode == DispModeText {
			for buf[chunkPos] == '\r' || buf[chunkPos] == '\n' {
				chunkPos++
				curLineOffset++
			}
			nlPos := bytes.IndexAny(buf[chunkPos:], "\r\n")
			if nlPos == -1 {
				nlPos = maxTextCols
			}
			chunks[c] = buf[chunkPos : chunkPos+int64(nlPos)]
		} else {
			chunks[c] = buf[chunkPos : chunkPos+cols]
		}

		if !g_dedup || dispMode == DispModeText || !bytes.Equal(chunks[c], chunks[1-c]) {
			// no dedup for this line or at all
			if was_separator {
				was_separator = false
				curSkip.end = curLineOffset
				skipMap[curSkip] = true
				curSkip = Range{}
			}

			drawLine(iLine, chunks[c], curLineOffset)

			iLine++
			curLineOffset += int64(len(chunks[c]))
			chunkPos += int64(len(chunks[c]))

			if dispMode == DispModeText {
				for chunkPos < int64(len(buf)) && (buf[chunkPos] == '\r' || buf[chunkPos] == '\n') {
					chunkPos++
					curLineOffset++
				}
			}

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
			curLineOffset += cols
			chunkPos += cols
		}

		if chunkPos >= int64(nRead) || (dispMode == DispModeText && chunkPos+int64(maxTextCols) >= int64(nRead)) {
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

func invalidateSkips() {
	skipMap = make(map[Range]bool)
}

func lastPageOffset() int64 {
	add := offset % cols
	return max(0, fileSize-fileSize%cols-int64(maxLinesPerPage-1)*cols+add)
}

func gotoOffset(new_offset int64) {
	breadcrumbs = append(breadcrumbs, Breadcrumb{offset, -1})
	offset = (new_offset - base) / baseMult
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

func setBookmark(n int) {
	bookmarks[n] = offset
}

func gotoBookmark(n int) {
	offset = bookmarks[n]
}

func calcDefaultCols() {
	scrWidth, _ := screen.Size()
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

func printLastErr() {
	if lastErrMsg != "" {
		fmt.Println(lastErrMsg)
	}
}

func shortenFName(fname string, max_len int) string {
	if utf8.RuneCountInString(fname) <= max_len {
		return fname
	}

	// find last '/' or '\'
	lastSlash := strings.LastIndexAny(fname, "/\\")
	if lastSlash != -1 {
		fname = fname[lastSlash+1:]
	}

	if utf8.RuneCountInString(fname) > max_len {
		fname = "…" + fname[utf8.RuneCountInString(fname)-max_len-1:]
	}
	return fname
}

func getAppDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "h"), nil
}

func main() {
	processFlags()

	file, err := os.Open(fname)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	if strings.HasPrefix(fname, "\\\\.\\PhysicalDrive") {
		fileSize, err = getDeviceSize(fname)
		if err != nil {
			panic(err)
		}
		reader = NewAlignedReader(file, fileSize, 512)
	} else {
		reader = file
		if isBlockDevice(fname) {
			fileSize, err = getDeviceSize(fname)
			if err != nil {
				panic(err)
			}
		} else {
			fileInfo, err := file.Stat()
			if err != nil {
				panic(err)
			}
			fileSize = fileInfo.Size()
		}
	}

	go initSearchHistory()
	go initCommandHistory()

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

	setCols(cols) // calc defaults if cols == 0
	//defaultColsMode = 1 // next mode

	go buildSparseMap()

	draw()
	handleEvents()
}
