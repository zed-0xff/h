package main

import (
	"bufio"
	"bytes"
	"strings"
)

const bufSize = 2 * 1024 * 1024

var (
	pattern []byte = make([]byte, 16)
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
	pattern_str := strings.TrimSpace(toHex(pattern, int64(scrWidth/3), 1))
	pattern_str = askBinStr("/", pattern_str)
	if pattern_str != "" {
		pattern = fromHex(pattern_str)
		searchNext()
	}
}

func searchPrev() {
	buffer := make([]byte, bufSize)
	patLen := len(pattern)
	window := make([]byte, 0)

	if offset <= 0 {
		return
	}
	newOffset := offset - bufSize + int64(patLen) - 1
	if newOffset < 0 {
		newOffset = 0
	}

	for {
		nRead, err := reader.ReadAt(buffer, newOffset)
		if nRead > 0 {
			if newOffset+int64(nRead) > offset+int64(patLen) {
				nRead = int(offset - newOffset + int64(patLen) - 1)
			}

			window = append(buffer[:nRead], window...)
			index := bytes.LastIndex(window, pattern)

			if index != -1 {
				newOffset += int64(index)
				if newOffset < offset {
					offset = newOffset
				}
				return
			}

			// Shrink the window to avoid unbounded growth
			if len(window) > patLen {
				window = window[:patLen]
			}
		}

		// Break if we've reached the beginning of the file
		if err != nil {
			return
		}

		if newOffset == 0 {
			return
		}

		// Update the new offset (move backward)
		newOffset -= int64(nRead)
		if newOffset < 0 {
			newOffset = 0
		}
	}
}

func searchNext() {
	buffer := make([]byte, bufSize)
	patLen := len(pattern)
	window := make([]byte, 0)

	newOffset := offset + 1
	reader.Seek(newOffset, 0)
	reader := bufio.NewReader(reader)
	for newOffset < fileSize && !checkInterrupt() {
		newOffset = findNextData(newOffset) // skip sparse regions
		updateProgress(newOffset)

		// Read a chunk of data from the reader
		n, err := reader.Read(buffer)
		if n > 0 {
			window = append(window, buffer[:n]...)

			// Search for the pattern in the current window
			index := bytes.Index(window, pattern)
			if index != -1 {
				offset = newOffset + int64(index)
				return
			}

			// Shrink the window to avoid unbounded growth
			if len(window) > patLen {
				window = window[len(window)-patLen:]
			}
		}

		// Break the loop if EOF is reached
		if err != nil {
			return
		}

		// Update the local offset
		newOffset += int64(n)
	}
}
