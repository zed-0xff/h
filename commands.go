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
	{"basemult", &baseMult, 16},
	{"pagesize", &pageSize, 10},
}

var COMMANDS = []struct {
	name string
	fn   func(string)
}{
	{"beep", func(string) { beep() }},
	{"goto", cmd_goto},
	{"set", cmd_set},
}

func cmd_goto(args string) {
	if args == "" {
		showErrStr("goto: need one argument")
		return
	}

	a := strings.Split(args, " ")
	if len(a) != 1 {
		showErrStr("goto: need one argument, got ", len(a))
		return
	}

	args = strings.TrimSpace(a[0])
	if args == "" {
		showErrStr("goto: empty argument")
		return
	}

	offs, err := parseExprRadix(args, 16)
	if err != nil {
		showError(err)
		return
	}
	gotoOffset(offs)
}

func cmd_set(args string) {
	if args == "" {
		// TODO: show current vars
		return
	}

	a := strings.Split(args, " ")
	for _, v := range a {
		args := strings.SplitN(v, "=", 2)
		if len(args) < 2 {
			showErrStr("set: need two arguments, got ", len(args))
			return
		}
		for i := range args {
			args[i] = strings.TrimSpace(args[i])
		}
		try_set_var(args[0], args[1])
	}
}

func run_cmd(cmd string) {
	a := strings.SplitN(cmd, " ", 2)
	cmd = a[0]

	args := ""
	if len(a) > 1 {
		args = a[1]
	}

	names := make([]string, 0)
	var pfun func(string)

	for _, c := range COMMANDS {
		if strings.HasPrefix(c.name, cmd) {
			names = append(names, c.name)
			pfun = c.fn
		}
	}

	switch len(names) {
	case 0:
		showErrStr("unknown command: " + cmd)
	case 1:
		pfun(args)
	default:
		showErrStr("ambiguous command: "+cmd+" (", strings.Join(names, ", "), ")")
	}
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
