package main

import (
	"context"
	"flag"
	"fmt"
	fileSystem "io/fs"
	"log"
	"os"
	"strings"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/absto/pkg/model"
)

func main() {
	fs := flag.NewFlagSet("absto", flag.ExitOnError)

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

	if fsApp, ok := storage.(fileSystem.FS); ok {
		err := fileSystem.WalkDir(fsApp, "pkg", func(path string, d fileSystem.DirEntry, err error) error {
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("fs", path)
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}

		// fileSystem.WalkDir(os.DirFS("/Users/macbook/code/absto"), "pkg", func(path string, d fileSystem.DirEntry, err error) error {
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}

		// 	fmt.Println("native", path)
		// 	return nil
		// })
	}
}
