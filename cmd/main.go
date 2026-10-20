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
	flagSet := flag.NewFlagSet("absto", flag.ExitOnError)
	flagSet.Usage = flags.Usage(flagSet)

	config := absto.Flags(flagSet, "")

	_ = flagSet.Parse(os.Args[1:])

	ctx := context.Background()

	storage, err := absto.New(config, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Creating a file `/test/first.txt`")
	log.Println(storage.WriteTo(ctx, "/test/first.txt", strings.NewReader("Streamed content"), model.WriteOpts{}))

	log.Println("Creating a file `/test/sub/second.txt`")
	log.Println(storage.WriteTo(ctx, "/test/sub/second.txt", strings.NewReader("Fixed size content"), model.WriteOpts{Size: 18}))

	log.Println("Renaming `/test` to `/renamed`")
	log.Println(storage.Rename(ctx, "/test", "/renamed"))

	log.Println("Walking `/renamed`")
	err = storage.Walk(ctx, "/renamed", func(item model.Item) error {
		if item.IsDir() {
			return nil
		}

		log.Println("Reading " + item.Pathname)
		reader, err := storage.ReadFrom(ctx, item.Pathname)
		log.Println(err)

		content, err := io.ReadAll(reader)
		log.Printf("`%s` : %s\n", content, err)

		return nil
	})
	log.Println(err)

	log.Println("Rename first to new")
	log.Println(storage.Rename(ctx, "/renamed/first.txt", "/new/test.txt"))

	log.Println("Remove /renamed")
	log.Println(storage.RemoveAll(ctx, "/renamed"))

	log.Println("Remove /new")
	log.Println(storage.RemoveAll(ctx, "/new"))
}
