package main

import (
	"strings"
)

func run_cmd(cmd string) bool {
	a := strings.SplitN(cmd, " ", 2)

	switch a[0] {
	case "beep":
		beep()
	case "set":
		return cmd_set(cmd)
	default:
		showErrStr("unknown command: " + a[0])
		return false
	}

	return true
}

func cmd_set(cmd string) bool {
	args := strings.SplitN(cmd, " ", 3)[1:]
	if len(args) < 2 {
		showErrStr("set: need two arguments, got ", len(args))
		return false
	}

	switch args[0] {
	case "cols":
		if val, err := parseExpr(args[1]); err == nil {
			setCols(val)
		} else {
			showError(err)
			return false
		}
	}

	return true
}
