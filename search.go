package main

import (
	"bufio"
	"bytes"
	"strings"
)

const bufSize = 8 * 1024 * 1024

var (
	g_searchPattern []byte = make([]byte, 0)
)

func hexDigitToInt(hexDigit byte) byte {
	if hexDigit >= '0' && hexDigit <= '9' {
		return hexDigit - '0'
	}
	if hexDigit >= 'A' && hexDigit <= 'F' {
		return hexDigit - 'A' + 10
	}
	if hexDigit >= 'a' && hexDigit <= 'f' {
		return hexDigit - 'a' + 10
	}
	return 0
}

func fromHex(hexStr string) []byte {
	hexStr = strings.Replace(hexStr, " ", "", -1)
	hexStr = strings.Replace(hexStr, "\n", "", -1)
	hexStr = strings.Replace(hexStr, "\r", "", -1)
	hexStr = strings.Replace(hexStr, "\t", "", -1)

	hexStr = strings.TrimPrefix(hexStr, "0x")
	hexStr = strings.TrimPrefix(hexStr, "0X")

	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	hexStr = strings.ToUpper(hexStr)

	n := len(hexStr) / 2
	bytes := make([]byte, n)
	for i := 0; i < n; i++ {
		bytes[i] = byte((hexDigitToInt(hexStr[i*2]) << 4) | hexDigitToInt(hexStr[i*2+1]))
	}

	return bytes
}

func searchUI() {
	newPattern := askSearchPattern(g_searchPattern)
	if newPattern != nil && len(newPattern) > 0 {
		g_searchPattern = newPattern
		searchNext()
	}
}

func searchPrev() bool {
	buf := make([]byte, bufSize)
	patLen := len(g_searchPattern)
	window := make([]byte, 0)

	if offset <= 0 {
		return false
	}
	newOffset := offset - bufSize + int64(patLen) - 1
	if newOffset < 0 {
		newOffset = 0
	}

	resetProgress()
	for {
		if checkInterrupt() {
			return true // don't beep
		}

		skipOffset := findPrevData(newOffset + bufSize) // skip sparse regions
		if skipOffset != -1 {
			newOffset = skipOffset - bufSize
			if newOffset < 0 {
				newOffset = 0
			}
			window = make([]byte, 0)
		}
		updateProgress(newOffset)

		nRead, _ := reader.ReadAt(buf, newOffset)
		if nRead > 0 {
			if newOffset+int64(nRead) > offset+int64(patLen) {
				nRead = int(offset - newOffset + int64(patLen) - 1)
			}

			window = append(buf[:nRead], window...)
			index := bytes.LastIndex(window, g_searchPattern)

			if index != -1 {
				newOffset += int64(index)
				if newOffset < offset {
					offset = newOffset
				}
				return true
			}

			// Shrink the window to avoid unbounded growth
			if len(window) > patLen {
				window = window[:patLen]
			}
		} else {
			return false
		}

		if newOffset == 0 {
			return false
		}

		// Update the new offset (move backward)
		newOffset -= int64(nRead)
		if newOffset < 0 {
			newOffset = 0
		}
	}
}

func searchNext() bool {
	buf := make([]byte, bufSize)
	patLen := len(g_searchPattern)
	window := make([]byte, 0)

	newOffset := offset + 1
	reader.Seek(newOffset, 0)
	scanner := bufio.NewReader(reader)
	resetProgress()
	for newOffset < fileSize {
		if checkInterrupt() {
			return true // don't beep
		}

		skipOffset := findNextData(newOffset) // skip sparse regions
		if skipOffset != -1 {
			newOffset = skipOffset
			reader.Seek(newOffset, 0)
			scanner.Reset(reader)
			window = make([]byte, 0)
		}
		updateProgress(newOffset)

		// Read a chunk of data from the reader
		n, err := scanner.Read(buf)
		if n > 0 {
			window = append(window, buf[:n]...)

			// Search for the pattern in the current window
			index := bytes.Index(window, g_searchPattern)
			if index != -1 {
				offset = newOffset + int64(index) - int64(len(window)) + int64(n)
				return true
			}

			// Shrink the window to avoid unbounded growth
			if len(window) > patLen {
				window = window[len(window)-patLen:]
			}
		}

		// Break the loop if EOF is reached
		if err != nil {
			return false
		}

		// Update the local offset
		newOffset += int64(n)
	}

	return false
}
