package main

import (
	"strings"
)

func run_cmd(cmd string) {
	a := strings.SplitN(cmd, " ", 2)

	switch a[0] {
	case "beep":
		beep()
	case "set":
		cmd_set(cmd)
	default:
		showErrStr("unknown command: " + a[0])
	}
}

func cmd_set(cmd string) {
	args := strings.SplitN(cmd, " ", 3)[1:]
	if len(args) < 2 {
		showErrStr("set: need two arguments, got ", len(args))
		return
	}

	switch args[0] {
	case "cols":
		if val, err := parseExpr(args[1]); err == nil {
			setCols(val)
		} else {
			showError(err)
		}
	}
}
