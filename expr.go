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

func parseExprRadix(expr string, radix int) (int64, error) {
	expr = strings.TrimSpace(expr)

	for order := 0; order < 14; order++ {
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
	if strings.HasPrefix(expr, "0x") || strings.HasPrefix(expr, "0X") {
		radix = 0
	}
	return strconv.ParseInt(expr, radix, 64)
}

func parseExpr(expr string) (int64, error) {
	return parseExprRadix(expr, 0)
}
