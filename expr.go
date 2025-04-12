package main

import (
	"strconv"
	"strings"
)

var OPS = []struct {
	order int
	op    byte
	fn    func(a, b int64) int64
}{
	// https://en.cppreference.com/w/cpp/language/operator_precedence
	{05, '*', func(a, b int64) int64 { return a * b }},
	{05, '/', func(a, b int64) int64 { return a / b }},
	{05, '%', func(a, b int64) int64 { return a % b }},

	{06, '+', func(a, b int64) int64 { return a + b }},
	{06, '-', func(a, b int64) int64 { return a - b }},

	{11, '&', func(a, b int64) int64 { return a & b }},
	{12, '^', func(a, b int64) int64 { return a ^ b }},
	{13, '|', func(a, b int64) int64 { return a | b }},
}

// used in ui.go
var EXPR_ALLOWED_CHARS = "0123456789abcdefxABCDEFX $" + func() string {
	seen := make(map[byte]bool)
	var ops []byte
	for _, op := range OPS {
		if !seen[op.op] {
			seen[op.op] = true
			ops = append(ops, op.op)
		}
	}
	return string(ops)
}()

var MAX_ORDER = func() int {
	maxOrder := 0
	for _, op := range OPS {
		if op.order > maxOrder {
			maxOrder = op.order
		}
	}
	return maxOrder
}()

var MIN_ORDER = func() int {
	minOrder := MAX_ORDER
	for _, op := range OPS {
		if op.order < minOrder {
			minOrder = op.order
		}
	}
	return minOrder
}()

// expects expr to be lowercase
func parseExprRadix_(expr string, radix int) (int64, error) {
	expr = strings.TrimSpace(expr)

	for order := MAX_ORDER; order >= MIN_ORDER; order-- {
		for _, op := range OPS {
			if op.order != order {
				continue
			}

			for i := 0; i < len(expr); i++ {
				if expr[i] == op.op {
					left, err := parseExprRadix(expr[:i], radix)
					if err != nil {
						return 0, err
					}
					right, err := parseExprRadix(expr[i+1:], radix)
					if err != nil {
						return 0, err
					}
					return op.fn(left, right), nil
				}
			}
		}
	}

	expr = strings.TrimSpace(expr)

	if expr == "$" {
		return here(), nil
	}
	if strings.HasPrefix(expr, "0") && len(expr) > 2 {
		switch expr[0:2] {
		case "0x", "0o", "0b": // golang's supported prefixes
			radix = 0
		case "0n": // 0n123 - force decimal
			radix = 10
			expr = expr[2:]
		}
	}
	return strconv.ParseInt(expr, radix, 64)
}

func parseExprRadix(expr string, radix int) (int64, error) {
	return parseExprRadix_(strings.ToLower(expr), radix)
}

func parseExpr(expr string) (int64, error) {
	return parseExprRadix(expr, 0)
}
