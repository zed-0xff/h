package main

import (
	"encoding/binary"
	"unicode/utf16"
	"unicode/utf8"
)

func decodeUTF16Bytes(input []byte, byteOrder binary.ByteOrder, replacement rune) string {
	var output []byte
	for i := 0; i+1 < len(input); {
		// Read one UTF-16 code unit
		codeUnit := byteOrder.Uint16(input[i : i+2])
		i += 2

		// Check for surrogate pair
		if utf16.IsSurrogate(rune(codeUnit)) {
			if i+1 >= len(input) {
				// No room for a second code unit
				output = appendRune(output, replacement)
				break
			}
			nextCodeUnit := byteOrder.Uint16(input[i : i+2])
			if r := utf16.DecodeRune(rune(codeUnit), rune(nextCodeUnit)); r != utf8.RuneError {
				output = appendRune(output, r)
				i += 2
			} else {
				// Invalid surrogate pair
				output = appendRune(output, replacement)
			}
		} else {
			output = appendRune(output, rune(codeUnit))
		}
	}

	return string(output)
}

func appendRune(dst []byte, r rune) []byte {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	return append(dst, buf[:n]...)
}

func decodeUTF16LE(input []byte) string {
	return decodeUTF16Bytes(input, binary.LittleEndian, '·')
}

func decodeUTF16BE(input []byte) string {
	return decodeUTF16Bytes(input, binary.BigEndian, '·')
}
