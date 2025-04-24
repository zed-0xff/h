package main

import (
	"fmt"
	"reflect"
	"strings"
)

var INT_VARS = []struct {
	name         string
	pvar         interface{}
	type_id      reflect.Type
	defaultRadix int
}{
	{"cols", &cols, reflect.TypeOf(cols), 10},
	{"base", &base, reflect.TypeOf(base), 16},
	{"baseMult", &baseMult, reflect.TypeOf(baseMult), 16},
	{"pageSize", &pageSize, reflect.TypeOf(pageSize), 10},
	{"allowWrite", &allowWrite, reflect.TypeOf(allowWrite), 0},
}

var COMMANDS = []struct {
	name string
	fn   func(string)
}{
	{"beep", func(string) { beep() }},
	{"goto", cmd_goto},
	{"print", cmd_print},
	{"set", cmd_set},
}

func cmd_print(args string) {
	if args == "" {
		showErrStr("print: need one argument")
		return
	}

	a := strings.Split(args, " ")
	if len(a) != 1 {
		showErrStr("print: need one argument, got ", len(a))
		return
	}

	args = strings.TrimSpace(a[0])
	if args == "" {
		showErrStr("print: empty argument")
		return
	}

	res, err := parseExprRadix(args, 16)
	if err != nil {
		showError(err)
		return
	}
	showMsg(fmt.Sprintf("0x%x (%d)", res, res))
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
		if strings.EqualFold(v.name, name) {
			switch p := v.pvar.(type) {
			case *int64:
				if val, err := parseExprRadix(expr, v.defaultRadix); err == nil {
					*p = val
				} else {
					showError(err)
					return false
				}
			case *bool:
				switch strings.ToLower(expr) {
				case "true", "1", "yes", "y":
					*p = true
				case "false", "0", "no", "n":
					*p = false
				default:
					showErrStr("invalid boolean value: ", expr)
					return false
				}
			default:
				showErrStr("unknown type for variable: ", name)
				return false
			}
			return true
		}
	}
	showErrStr("unknown variable: " + name)
	return false
}
