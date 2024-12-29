//go:build !windows
package main

import (
	"errors"
)

func getDriveSize(drive string) (int64, error) {
    return 0, errors.New("Not implemented")
}
