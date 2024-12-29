package main

import (
	"bytes"
    "fmt"
    "strconv"
    "strings"

	"github.com/nsf/termbox-go"
)

func ask(prompt, curValue, allowedChars string) string {
    printAt(0, maxLinesPerPage, prompt)
    termbox.Flush()

    buffer := bytes.NewBufferString(curValue)
    for {
        printAt(0, maxLinesPerPage, prompt + buffer.String())
        x := len(prompt) + buffer.Len()
        termbox.SetCell(x, maxLinesPerPage, ' ', termbox.AttrReverse, termbox.ColorDefault)
        termbox.SetCell(x+1, maxLinesPerPage, ' ', termbox.ColorDefault, termbox.ColorDefault)
        termbox.Flush()

        switch ev := termbox.PollEvent(); ev.Type {
        case termbox.EventKey:
            switch ev.Key {
            case termbox.KeyEsc, termbox.KeyCtrlC:
                return ""
            case termbox.KeyEnter:
                return buffer.String()
            case termbox.KeyBackspace, termbox.KeyBackspace2:
                if buffer.Len() > 0 {
                    buffer.Truncate(buffer.Len() - 1)
                }
            default:
                c := ev.Ch
                if ev.Key == termbox.KeySpace {
                    c = ' '
                }
                if len(allowedChars) == 0 || strings.Contains(allowedChars, string(c)) {
                    buffer.WriteRune(c)
                } else {
                    beep()
                }
            }
        }
    }
}

func beep(){
    fmt.Print("\a")
}

func askString(prompt, curValue string) string {
    return ask(prompt, curValue, "")
}

func askBinStr(prompt, curValue string) string {
    return ask(prompt, curValue, "0123456789abcdefABCDEF ")
}

func askInt(prompt string, curValue int64) int64 {
    s := ask(prompt, fmt.Sprintf("%d", curValue), "0123456789abcdefxABCDEFX")
    if s == "" {
        return curValue
    }
    n, err := strconv.ParseInt(s, 0, 64)
    if err != nil {
        beep()
        return curValue
    }
    return n
}

func askHexInt(prompt string, curValue int64) int64 {
    s := ask(prompt, fmt.Sprintf("%x", curValue), "0123456789abcdefxABCDEFX")
    if s == "" {
        return curValue
    }
    n, err := strconv.ParseInt(s, 16, 64)
    if err != nil {
        beep()
        return curValue
    }
    return n
}
