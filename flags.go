package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

func processFlags() {
	pflag.Int64Var(&pageSize, "page-size", 0, "custom page size for pgup/pgdn (default: auto)")
	pflag.Int64Var(&cols, "cols", 0, "number of columns to display (default: auto)")
	pflag.BoolVarP(&g_dedup, "dedup", "d", true, "deduplicate output")

	pflag.BoolVarP(&showBin, "binary", "B", false, "show binary representation")
	pflag.BoolVarP(&showHex, "hex", "H", true, "show hexadecimal representation")
	pflag.BoolVarP(&showASCII, "ascii", "A", true, "show ASCII representation")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <filename> [offset]\n", os.Args[0])
		pflag.PrintDefaults()
	}

	pflag.Parse()

	println("dedup:", g_dedup)

	pos_args := pflag.Args()
	if len(pos_args) == 0 {
		pflag.Usage()
		os.Exit(1)
	}

	fname = pos_args[0]

	if len(pos_args) > 1 {
		//		for i := 0; i < len(os.Args); i++ {
		//			if os.Args[i] == "--debug" {
		//				os.Args = append(os.Args[:i], os.Args[i+1:]...)
		//				fmt.Println("Size:", fileSize)
		//				buildSparseMap()
		//				fmt.Println("Sparse map:")
		//				for i, r := range sparseMap {
		//					fmt.Printf("%2x: %12x %12x\n", i, r.start, r.end)
		//				}
		//				os.Exit(0)
		//			}
		//		}

		var err error
		str := strings.TrimPrefix(strings.ToLower(pos_args[1]), "0x")
		offset, err = strconv.ParseInt(str, 16, 64)
		if err != nil {
			fmt.Println("Error parsing offset:", err)
			os.Exit(1)
		}
	}

}
