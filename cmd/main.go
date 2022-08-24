package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/absto/pkg/model"
)

func main() {
	fs := flag.NewFlagSet("absto", flag.ExitOnError)

	config := absto.Flags(fs, "")

	fmt.Println(fs.Parse(os.Args[1:]))

	storage, err := absto.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}

	items, err := storage.List(context.Background(), "")
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		fmt.Printf("%+v\n", item)
	}

	fmt.Println(storage.CreateDir(context.Background(), "/test"))
	fmt.Println(storage.WriteTo(context.Background(), "/test/example.txt", strings.NewReader("Empty content"), model.WriteOpts{}))
	fmt.Println(storage.WriteTo(context.Background(), "/test/second.txt", strings.NewReader("Empty content second"), model.WriteOpts{}))
	fmt.Println(storage.Rename(context.Background(), "/test/", "/renamed/"))
	fmt.Println(storage.Rename(context.Background(), "/renamed/example.txt", "/new/test.txt"))
}
