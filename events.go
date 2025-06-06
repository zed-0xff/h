package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

func handleEvents() {
	for {
		dir := 0
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			scrWidth, scrHeight = ev.Size()
			if !customColsMode {
				calcDefaultCols(scrWidth)
			}
			draw()

		case *tcell.EventKey:
			lastErrMsg = "" // reset last error message on any key event

			switch ev.Key() {
			case tcell.KeyLeft:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				dir = -1
				if ev.Modifiers() == tcell.ModShift {
					offset -= 1
				} else {
					offset -= int64(elWidth)
				}
				invalidateSkips()
			case tcell.KeyRight:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				dir = 1
				if ev.Modifiers() == tcell.ModShift {
					offset += 1
				} else {
					offset += int64(elWidth)
				}
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
				new_offset := askOffset("[hex] offset: ", offset)
				if new_offset != offset {
					gotoOffset(new_offset)
				}
			case tcell.KeyPgDn:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyPgDn})
				dir = 1
				if pageSize == 0 {
					offset = nextOffset
				} else {
					offset += pageSize
				}
			case tcell.KeyPgUp:
				// efficiently handle skipping over deduplicated lines
				if len(breadcrumbs) > 0 && breadcrumbs[len(breadcrumbs)-1].key == tcell.KeyPgDn {
					offset = breadcrumbs[len(breadcrumbs)-1].offset
					breadcrumbs = breadcrumbs[:len(breadcrumbs)-1]
				} else {
					breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyPgUp})
					if pageSize == 0 {
						offset -= cols * int64(maxLinesPerPage)
					} else {
						offset -= pageSize
					}
				}
				dir = -1
			case tcell.KeyHome:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				offset = 0
			case tcell.KeyEnd:
				breadcrumbs = append(breadcrumbs, Breadcrumb{offset, ev.Key()})
				offset = lastPageOffset()
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(breadcrumbs) > 0 {
					offset = breadcrumbs[len(breadcrumbs)-1].offset
					breadcrumbs = breadcrumbs[:len(breadcrumbs)-1]
				}
			case tcell.KeyTab, tcell.KeyEnter:
				dispMode += 1
				if dispMode > DispModeMax {
					dispMode = 0
				}
			case tcell.KeyEsc, tcell.KeyCtrlC:
				return

			case tcell.KeyRune:
				if ev.Modifiers() == tcell.ModAlt {
					switch ev.Rune() {
					case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
						gotoBookmark(int(ev.Rune() - '0'))
					}

				} else {
					// no modifiers

					switch ev.Rune() {
					case '!':
						setBookmark(1)
					case '@':
						setBookmark(2)
					case '#':
						setBookmark(3)
					case '$':
						setBookmark(4)
					case '%':
						setBookmark(5)
					case '^':
						setBookmark(6)
					case '&':
						setBookmark(7)
					case '*':
						setBookmark(8)
					case '(':
						setBookmark(9)
					case ')':
						setBookmark(0)

					case ' ':
						breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyPgDn})
						dir = 1
						if pageSize == 0 {
							offset = nextOffset
						} else {
							offset += pageSize
						}
					case '-':
						customColsMode = true
						defaultColsMode = 1
						if cols-int64(elWidth) > 0 {
							cols -= int64(elWidth)
							invalidateSkips()
						}
					case '_':
						customColsMode = true
						defaultColsMode = 1
						if cols > 1 {
							cols /= 2
							invalidateSkips()
						}
					case '=':
						customColsMode = true
						defaultColsMode = 1
						cols += int64(elWidth)
						invalidateSkips()
					case '+':
						customColsMode = true
						defaultColsMode = 1
						cols *= 2
						invalidateSkips()
					case '0':
						customColsMode = false
						defaultColsMode = 1 - defaultColsMode
						calcDefaultCols(scrWidth)
					case '1', '2', '4', '8':
						elWidth = int(ev.Rune() - '0')
					case ':':
						cmd := askCommand()
						if cmd != "" {
							run_cmd(cmd)
						}
					case 'a':
						showASCII = !showASCII
					case 'b':
						showBin = !showBin
					case 'B':
						binMode01 = !binMode01
					case 'C':
						altColorMode = !altColorMode
					case 'c', 'w':
						setCols(askInt("cols: ", cols))
					case 'h':
						showHex = !showHex
					case '9':
						elWidth = 0x10
					case 'd':
						g_dedup = !g_dedup
					case 'g':
						new_offset := askOffset("[hex] offset: ", offset)
						if new_offset != offset {
							gotoOffset(new_offset)
						}
						//breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyHome})
						//offset = 0
					case 'G':
						breadcrumbs = append(breadcrumbs, Breadcrumb{offset, tcell.KeyEnd})
						offset = lastPageOffset()
					case 'n':
						if !searchNext() {
							beep()
						}
					case 'N':
						if !searchPrev() {
							beep()
						}
					case 'p': // set page size
						pageSize = askInt("page size (0 = auto): ", pageSize)
					case 'P': // patch file
						if allowWrite {
							patchOffset := askHexInt("[hex] offset: ", offset)
							if patchOffset >= 0 {
								patchSize := askHexInt("[hex] size: ", 0)
								if patchSize > 0 {
									patchData := askPattern("data ", []byte{0})
									if len(patchData) > 0 {
										if len(patchData) <= int(patchSize) {
											patchFile(patchOffset, patchSize, patchData)
										} else {
											showErrStr("data too long")
										}
									}
								}
							}
						} else {
							showErrStr("Writing is not allowed (hint: use -w option or ':set allowWrite=1')")
						}
					case 'q', 'Q':
						return
					case 'u':
						showUnicode = !showUnicode
					case 'U':
						unicodeMode = !unicodeMode
					case 'W':
						fname := askString("write to: ", fmt.Sprintf("%0*x.bin", offsetWidth, here()))
						if fname != "" {
							size := askHexInt("[hex] size: ", 0x1000)
							if size > 0 {
								err := writeFile(fname, offset, size)
								if err != nil {
									beep()
								}
							}
						}
					case '/', '?':
						searchUI(ev.Rune() == '/')
					}
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
			} else if offset > fileSize {
				offset = fileSize
			}
			draw()
		}
	}
}
