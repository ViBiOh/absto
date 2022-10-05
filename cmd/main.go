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

	fmt.Println(storage.CreateDir(context.Background(), "/test"))
	fmt.Println(storage.WriteTo(context.Background(), "/test/example.txt", strings.NewReader("Streamed content"), model.WriteOpts{}))
	fmt.Println(storage.WriteTo(context.Background(), "/test/second.txt", strings.NewReader("Fixed size content"), model.WriteOpts{Size: 18}))

	fmt.Println(storage.Rename(context.Background(), "/test/", "/renamed/"))

	items, err := storage.List(context.Background(), "/renamed/")
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		fmt.Printf("%+v\n", item)
	}

	if len(items) != 2 {
		log.Fatal("too many files in bucket")
	}

	fmt.Println(storage.Rename(context.Background(), "/renamed/example.txt", "/new/test.txt"))

	fmt.Println(storage.Remove(context.Background(), "/renamed"))
	fmt.Println(storage.Remove(context.Background(), "/new"))
}
