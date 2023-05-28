package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/flags"
)

func main() {
	fs := flag.NewFlagSet("absto", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	config := absto.Flags(fs, "")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	storage, err := absto.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}

	var hasErr bool

	log.Println(storage.CreateDir(ctx, "/test"))
	log.Println(storage.WriteTo(ctx, "/test/example.txt", strings.NewReader("Streamed content"), model.WriteOpts{}))
	log.Println(storage.WriteTo(ctx, "/test/second.txt", strings.NewReader("Fixed size content"), model.WriteOpts{Size: 18}))

	log.Println(storage.Rename(ctx, "/test/", "/renamed/"))

	items, err := storage.List(ctx, "/renamed/")
	if err != nil {
		hasErr = true

		log.Println(err)
	}

	for _, item := range items {
		log.Printf("%+v\n", item)
	}

	if len(items) != 2 {
		hasErr = true

		log.Println("too many files in bucket")
	}

	log.Println(storage.Rename(ctx, "/renamed/example.txt", "/new/test.txt"))

	log.Println(storage.Remove(ctx, "/renamed"))
	log.Println(storage.Remove(ctx, "/new"))

	if hasErr {
		os.Exit(1)
	}
}
