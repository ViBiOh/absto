package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ViBiOh/absto/pkg/absto"
)

func main() {
	fs := flag.NewFlagSet("absto", flag.ExitOnError)

	absto.Flags(fs, "")

	fmt.Println(fs.Parse(os.Args[1:]))
}
