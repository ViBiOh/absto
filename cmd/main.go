package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ViBiOh/absto/pkg/absto"
)

func main() {
	fs := flag.NewFlagSet("absto", flag.ExitOnError)

	config := absto.Flags(fs, "")

	fmt.Println(fs.Parse(os.Args[1:]))

	storage, err := absto.New(config)
	if err != nil {
		log.Fatal(err)
	}

	items, err := storage.List("")
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		fmt.Printf("%+v\n", item)
	}
}
