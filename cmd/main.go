package main

import (
	"context"
	"flag"
	"io"
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

	log.Println("Creating directory `/test`")
	log.Println(storage.Mkdir(ctx, "/test", model.DirectoryPerm))

	log.Println("Creating a file `/test/first.txt`")
	log.Println(storage.WriteTo(ctx, "/test/first.txt", strings.NewReader("Streamed content"), model.WriteOpts{}))

	log.Println("Creating a file `/test/second.txt`")
	log.Println(storage.WriteTo(ctx, "/test/second.txt", strings.NewReader("Fixed size content"), model.WriteOpts{Size: 18}))

	log.Println("Open file `/test/third.txt`")

	file, err := storage.OpenFile(ctx, "/test/third.txt", model.WriteFlag, model.RegularFilePerm)
	log.Println(err)

	log.Println("Writing to `/test/third.txt`")
	log.Println(file.Write([]byte("Writer content")))

	log.Println("Closing `/test/third.txt`")
	log.Println(file.Close())

	log.Println("Renaming `/test/` to `/renamed/`")
	log.Println(storage.Rename(ctx, "/test/", "/renamed/"))

	log.Println("Listing `/renamed/`")
	items, err := storage.List(ctx, "/renamed/")
	log.Println(err)

	for _, item := range items {
		log.Println("Reading " + item.Pathname)
		reader, err := storage.ReadFrom(ctx, item.Pathname)
		log.Println(err)

		content, err := io.ReadAll(reader)
		log.Printf("`%s` : %s\n", content, err)
	}

	if len(items) != 3 {
		log.Println("not expected file in bucket")
	}

	log.Println(storage.Rename(ctx, "/renamed/first.txt", "/new/test.txt"))

	log.Println(storage.RemoveAll(ctx, "/renamed"))
	log.Println(storage.RemoveAll(ctx, "/new"))
}
