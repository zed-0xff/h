package main

import (
	"strings"
)

var INT_VARS = []struct {
	name         string
	pvar         *int64
	defaultRadix int
}{
	{"cols", &cols, 10},
	{"base", &base, 16},
	{"pagesize", &pageSize, 10},
}

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

func try_set_var(name, expr string) bool {
	for _, v := range INT_VARS {
		if v.name == name {
			if val, err := parseExprRadix(expr, v.defaultRadix); err == nil {
				*v.pvar = val
				return true
			} else {
				showError(err)
				return false
			}
		}
	}
	showErrStr("unknown variable: " + name)
	return false
}

func cmd_set(cmd string) bool {
	a := strings.SplitN(cmd, " ", 2)
	if len(a) < 2 {
		// TODO: show current vars
		return false
	}

	args := strings.SplitN(a[1], "=", 2)
	if len(args) < 2 {
		showErrStr("set: need two arguments, got ", len(args))
		return false
	}
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}

	return try_set_var(args[0], args[1])
}
