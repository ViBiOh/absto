package storagetest

import (
	"context"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
)

type Factory func(t *testing.T) model.Storage

func Run(t *testing.T, factory Factory) {
	t.Helper()

	suites := []func(*testing.T, Factory){
		testStat,
		testList,
		testWalk,
		testIgnoreFn,
		testMkdir,
		testWriteTo,
		testReadFrom,
		testRename,
		testRemoveAll,
	}

	for index, suite := range suites {
		t.Run("index "+strconv.Itoa(index), func(t *testing.T) {
			suite(t, factory)
		})
	}
}

func seed(t *testing.T, factory Factory) model.Storage {
	t.Helper()

	cleanup := clean(t, factory)
	cleanup()
	t.Cleanup(cleanup)

	storage := factory(t)
	ctx := context.Background()

	files := map[string]string{
		"/photo/a.jpg":     "a",
		"/photo/sub/b.jpg": "b",
		"/photos/c.jpg":    "c",
		"/photo.jpg":       "root",
	}

	for name, content := range files {
		if err := storage.WriteTo(ctx, name, strings.NewReader(content), model.WriteOpts{}); err != nil {
			t.Fatalf("seed WriteTo(%s) error: %s", name, err)
		}
	}

	if err := storage.Mkdir(ctx, "/empty", model.DirectoryPerm); err != nil {
		t.Fatalf("seed Mkdir() error: %s", err)
	}

	return storage
}

func clean(t *testing.T, factory Factory) func() {
	return func() {
		ctx := context.Background()
		storage := factory(t)

		items, err := storage.List(ctx, "/")
		if err != nil {
			t.Fatalf("clean List error: %s", err)
		}

		for _, item := range items {
			if err := storage.RemoveAll(ctx, item.Pathname); err != nil {
				t.Fatalf("clean RemoveAll(%s) error: %s", item.Pathname, err)
			}
		}
	}
}

func pathnames(items []model.Item) []string {
	output := make([]string, 0, len(items))

	for _, item := range items {
		output = append(output, item.Pathname)
	}

	return output
}

func walkAll(storage model.Storage, name string) ([]string, error) {
	var output []string

	err := storage.Walk(context.Background(), name, func(item model.Item) error {
		output = append(output, item.Pathname)

		return nil
	})

	return output, err
}

func assertPathnames(t *testing.T, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertNotExist(t *testing.T, err error) {
	t.Helper()

	if !model.IsNotExist(err) {
		t.Errorf("error = %v, want not exist", err)
	}
}

func testStat(t *testing.T, factory Factory) {
	storage := seed(t, factory)

	cases := map[string]struct {
		name      string
		wantPath  string
		wantName  string
		wantIsDir bool
	}{
		"file":                    {"/photo.jpg", "/photo.jpg", "photo.jpg", false},
		"nested file":             {"/photo/sub/b.jpg", "/photo/sub/b.jpg", "b.jpg", false},
		"directory":               {"/photo", "/photo", "photo", true},
		"directory with slash":    {"/photo/", "/photo", "photo", true},
		"implicit directory":      {"/photo/sub", "/photo/sub", "sub", true},
		"empty directory":         {"/empty", "/empty", "empty", true},
		"empty directory slash":   {"/empty/", "/empty", "empty", true},
		"relative style name":     {"photo/a.jpg", "/photo/a.jpg", "a.jpg", false},
		"directory without slash": {"/photos", "/photos", "photos", true},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			item, err := storage.Stat(context.Background(), tc.name)
			if err != nil {
				t.Fatalf("Stat() unexpected error: %s", err)
			}

			if item.Pathname != tc.wantPath || item.Name() != tc.wantName || item.IsDir() != tc.wantIsDir {
				t.Errorf("Stat() = %+v, want pathname `%s`, name `%s`, dir %t", item, tc.wantPath, tc.wantName, tc.wantIsDir)
			}

			if item.ID != model.ID(tc.wantPath) {
				t.Errorf("Stat() ID = %s, want %s", item.ID, model.ID(tc.wantPath))
			}
		})
	}

	t.Run("file size and extension", func(t *testing.T) {
		item, err := storage.Stat(context.Background(), "/photo.jpg")
		if err != nil {
			t.Fatal(err)
		}

		if item.Size() != 4 || item.Extension != ".jpg" {
			t.Errorf("Stat() = %+v, want size 4 and extension `.jpg`", item)
		}
	})

	t.Run("root", func(t *testing.T) {
		item, err := storage.Stat(context.Background(), "/")
		if err != nil {
			t.Fatal(err)
		}

		if item.Pathname != "/" || !item.IsDir() {
			t.Errorf("Stat(/) = %+v, want the root directory", item)
		}
	})

	for _, name := range []string{"/missing", "/missing/", "/photo.jpg/", "/photo/missing.jpg", "/photo.jpg/child"} {
		t.Run("not exist "+name, func(t *testing.T) {
			_, err := storage.Stat(context.Background(), name)
			assertNotExist(t, err)
		})
	}

	t.Run("relative path", func(t *testing.T) {
		if _, err := storage.Stat(context.Background(), "/photo/../photos"); err != model.ErrRelativePath {
			t.Errorf("Stat() error = %v, want %v", err, model.ErrRelativePath)
		}
	})
}

func testList(t *testing.T, factory Factory) {
	storage := seed(t, factory)

	cases := map[string][]string{
		"/photo":     {"/photo/a.jpg", "/photo/sub"},
		"/photo/":    {"/photo/a.jpg", "/photo/sub"},
		"/photo/sub": {"/photo/sub/b.jpg"},
		"/":          {"/empty", "/photo.jpg", "/photo", "/photos"},
		"":           {"/empty", "/photo.jpg", "/photo", "/photos"},
		"/empty":     nil,
		"/empty/":    nil,
		"photo/sub":  {"/photo/sub/b.jpg"},
		"/photos":    {"/photos/c.jpg"},
	}

	for name, want := range cases {
		t.Run("list `"+name+"`", func(t *testing.T) {
			items, err := storage.List(context.Background(), name)
			if err != nil {
				t.Fatalf("List() unexpected error: %s", err)
			}

			assertPathnames(t, pathnames(items), want)
		})
	}

	for _, name := range []string{"/missing", "/missing/", "/photo.jpg", "/photo.jpg/", "/photo/a.jpg", "/pho"} {
		t.Run("not exist "+name, func(t *testing.T) {
			_, err := storage.List(context.Background(), name)
			assertNotExist(t, err)
		})
	}
}

func testWalk(t *testing.T, factory Factory) {
	storage := seed(t, factory)

	cases := map[string][]string{
		"/photo":     {"/photo", "/photo/a.jpg", "/photo/sub", "/photo/sub/b.jpg"},
		"/photo/":    {"/photo", "/photo/a.jpg", "/photo/sub", "/photo/sub/b.jpg"},
		"/photo/sub": {"/photo/sub", "/photo/sub/b.jpg"},
		"/empty":     {"/empty"},
		"/": {
			"/",
			"/empty",
			"/photo.jpg",
			"/photo",
			"/photo/a.jpg",
			"/photo/sub",
			"/photo/sub/b.jpg",
			"/photos",
			"/photos/c.jpg",
		},
	}

	for name, want := range cases {
		t.Run("walk `"+name+"`", func(t *testing.T) {
			got, err := walkAll(storage, name)
			if err != nil {
				t.Fatalf("Walk() unexpected error: %s", err)
			}

			assertPathnames(t, got, want)
		})
	}

	for _, name := range []string{"/missing", "/missing/", "/pho", "/photo.jp"} {
		t.Run("not exist "+name, func(t *testing.T) {
			_, err := walkAll(storage, name)
			assertNotExist(t, err)
		})
	}

	t.Run("callback error stops the walk", func(t *testing.T) {
		var visited int

		err := storage.Walk(context.Background(), "/", func(model.Item) error {
			visited++

			return io.ErrUnexpectedEOF
		})
		if err != io.ErrUnexpectedEOF {
			t.Errorf("Walk() error = %v, want %v", err, io.ErrUnexpectedEOF)
		}

		if visited != 1 {
			t.Errorf("Walk() visited %d items, want 1", visited)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		if _, err := walkAll(storage, "/../photo"); err != model.ErrRelativePath {
			t.Errorf("Walk() error = %v, want %v", err, model.ErrRelativePath)
		}
	})
}

func testIgnoreFn(t *testing.T, factory Factory) {
	storage := seed(t, factory)

	ignoring := storage.WithIgnoreFn(func(item model.Item) bool {
		return item.Name() == "sub"
	})

	t.Run("list", func(t *testing.T) {
		items, err := ignoring.List(context.Background(), "/photo")
		if err != nil {
			t.Fatal(err)
		}

		assertPathnames(t, pathnames(items), []string{"/photo/a.jpg"})
	})

	t.Run("walk", func(t *testing.T) {
		got, err := walkAll(ignoring, "/photo")
		if err != nil {
			t.Fatal(err)
		}

		assertPathnames(t, got, []string{"/photo", "/photo/a.jpg"})
	})

	t.Run("mutations act on everything", func(t *testing.T) {
		mutable := seed(t, factory)
		mutableIgnoring := mutable.WithIgnoreFn(func(item model.Item) bool {
			return item.Name() == "sub"
		})

		ctx := context.Background()

		if err := mutableIgnoring.Rename(ctx, "/photo", "/img"); err != nil {
			t.Fatal(err)
		}

		got, err := walkAll(mutable, "/img")
		if err != nil {
			t.Fatal(err)
		}

		assertPathnames(t, got, []string{"/img", "/img/a.jpg", "/img/sub", "/img/sub/b.jpg"})

		if err = mutableIgnoring.RemoveAll(ctx, "/img"); err != nil {
			t.Fatal(err)
		}

		_, err = mutable.Stat(ctx, "/img")
		assertNotExist(t, err)
	})
}

func testMkdir(t *testing.T, factory Factory) {
	ctx := context.Background()

	t.Run("nested and idempotent", func(t *testing.T) {
		storage := seed(t, factory)

		for range 2 {
			if err := storage.Mkdir(ctx, "/a/b/c", model.DirectoryPerm); err != nil {
				t.Fatalf("Mkdir() unexpected error: %s", err)
			}
		}

		for _, name := range []string{"/a", "/a/b", "/a/b/c"} {
			item, err := storage.Stat(ctx, name)
			if err != nil || !item.IsDir() {
				t.Errorf("Stat(%s) = %+v, %v, want a directory", name, item, err)
			}
		}

		items, err := storage.List(ctx, "/a/b/c")
		if err != nil {
			t.Fatal(err)
		}

		assertPathnames(t, pathnames(items), nil)
	})

	t.Run("existing implicit directory", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.Mkdir(ctx, "/photo/sub", model.DirectoryPerm); err != nil {
			t.Fatalf("Mkdir() unexpected error: %s", err)
		}

		items, err := storage.List(ctx, "/photo/sub")
		if err != nil {
			t.Fatal(err)
		}

		assertPathnames(t, pathnames(items), []string{"/photo/sub/b.jpg"})
	})

	t.Run("file in the way", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.Mkdir(ctx, "/photo.jpg", model.DirectoryPerm); err == nil {
			t.Error("Mkdir() want an error when a file has the same name")
		}
	})

	for _, name := range []string{"/photo.jpg/child", "/photo/a.jpg/child"} {
		t.Run("through a file "+name, func(t *testing.T) {
			storage := seed(t, factory)

			err := storage.Mkdir(ctx, name, model.DirectoryPerm)
			if err == nil {
				t.Fatal("Mkdir() want an error when a file is in the way")
			}

			assertNotExist(t, err)
		})
	}
}

func testWriteTo(t *testing.T, factory Factory) {
	storage := seed(t, factory)
	ctx := context.Background()

	t.Run("creates missing parents", func(t *testing.T) {
		if err := storage.WriteTo(ctx, "/new/deep/file.txt", strings.NewReader("content"), model.WriteOpts{}); err != nil {
			t.Fatalf("WriteTo() unexpected error: %s", err)
		}

		for _, name := range []string{"/new", "/new/deep"} {
			item, err := storage.Stat(ctx, name)
			if err != nil || !item.IsDir() {
				t.Errorf("Stat(%s) = %+v, %v, want a directory", name, item, err)
			}
		}

		items, err := storage.List(ctx, "/new/deep")
		if err != nil {
			t.Fatal(err)
		}

		assertPathnames(t, pathnames(items), []string{"/new/deep/file.txt"})
	})

	t.Run("overwrites", func(t *testing.T) {
		if err := storage.WriteTo(ctx, "/photo.jpg", strings.NewReader("x"), model.WriteOpts{}); err != nil {
			t.Fatal(err)
		}

		reader, err := storage.ReadFrom(ctx, "/photo.jpg")
		if err != nil {
			t.Fatal(err)
		}

		defer func() { _ = reader.Close() }()

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "x" {
			t.Errorf("content = %q, want `x`", content)
		}
	})

	t.Run("directory name", func(t *testing.T) {
		if err := storage.WriteTo(ctx, "/photo/", strings.NewReader("x"), model.WriteOpts{}); err == nil {
			t.Error("WriteTo() want an error for a name ending with a slash")
		}
	})

	t.Run("relative path", func(t *testing.T) {
		if err := storage.WriteTo(ctx, "/../x", strings.NewReader("x"), model.WriteOpts{}); err != model.ErrRelativePath {
			t.Errorf("WriteTo() error = %v, want %v", err, model.ErrRelativePath)
		}
	})
}

func testReadFrom(t *testing.T, factory Factory) {
	storage := seed(t, factory)
	ctx := context.Background()

	t.Run("content", func(t *testing.T) {
		reader, err := storage.ReadFrom(ctx, "/photo/sub/b.jpg")
		if err != nil {
			t.Fatal(err)
		}

		defer func() { _ = reader.Close() }()

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "b" {
			t.Errorf("content = %q, want `b`", content)
		}
	})

	for _, name := range []string{"/missing", "/photo/missing.jpg", "/photos/c.jpg/child"} {
		t.Run("not exist "+name, func(t *testing.T) {
			reader, err := storage.ReadFrom(ctx, name)
			if err == nil {
				_ = reader.Close()
			}

			assertNotExist(t, err)
		})
	}
}

func testRename(t *testing.T, factory Factory) {
	ctx := context.Background()

	wantMoved := []string{"/img", "/img/a.jpg", "/img/sub", "/img/sub/b.jpg"}

	for _, names := range [][2]string{
		{"/photo", "/img"},
		{"/photo/", "/img/"},
		{"/photo/", "/img"},
		{"/photo", "/img/"},
		{"photo", "img"},
	} {
		t.Run("directory "+names[0]+" to "+names[1], func(t *testing.T) {
			storage := seed(t, factory)

			if err := storage.Rename(ctx, names[0], names[1]); err != nil {
				t.Fatalf("Rename() unexpected error: %s", err)
			}

			got, err := walkAll(storage, "/img")
			if err != nil {
				t.Fatal(err)
			}

			assertPathnames(t, got, wantMoved)

			_, err = storage.Stat(ctx, "/photo")
			assertNotExist(t, err)

			for _, name := range []string{"/photos/c.jpg", "/photo.jpg"} {
				if _, err = storage.Stat(ctx, name); err != nil {
					t.Errorf("sibling %s should be untouched: %s", name, err)
				}
			}
		})
	}

	t.Run("file with new parents", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.Rename(ctx, "/photo.jpg", "/moved/deep/p.jpg"); err != nil {
			t.Fatalf("Rename() unexpected error: %s", err)
		}

		if _, err := storage.Stat(ctx, "/moved/deep/p.jpg"); err != nil {
			t.Errorf("destination: %s", err)
		}

		_, err := storage.Stat(ctx, "/photo.jpg")
		assertNotExist(t, err)
	})

	t.Run("missing source", func(t *testing.T) {
		storage := seed(t, factory)

		assertNotExist(t, storage.Rename(ctx, "/missing", "/other"))
		assertNotExist(t, storage.Rename(ctx, "/pho", "/other"))

		if _, err := storage.Stat(ctx, "/other"); !model.IsNotExist(err) {
			t.Errorf("destination of a failed rename exists: %v", err)
		}
	})

	t.Run("into itself", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.Rename(ctx, "/photo", "/photo/inner"); err == nil {
			t.Error("Rename() want an error when moving a directory into itself")
		}

		if _, err := storage.Stat(ctx, "/photo/a.jpg"); err != nil {
			t.Errorf("source damaged: %s", err)
		}
	})

	t.Run("same path", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.Rename(ctx, "/photo", "/photo"); err != nil {
			t.Errorf("Rename() unexpected error: %s", err)
		}

		if _, err := storage.Stat(ctx, "/photo/a.jpg"); err != nil {
			t.Errorf("source damaged: %s", err)
		}
	})

	t.Run("root", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.Rename(ctx, "/", "/other"); err == nil {
			t.Error("Rename() want an error for the root")
		}
	})
}

func testRemoveAll(t *testing.T, factory Factory) {
	ctx := context.Background()

	for _, name := range []string{"/photo", "/photo/"} {
		t.Run("directory "+name, func(t *testing.T) {
			storage := seed(t, factory)

			if err := storage.RemoveAll(ctx, name); err != nil {
				t.Fatalf("RemoveAll() unexpected error: %s", err)
			}

			_, err := storage.Stat(ctx, "/photo")
			assertNotExist(t, err)

			got, err := walkAll(storage, "/")
			if err != nil {
				t.Fatal(err)
			}

			assertPathnames(t, got, []string{"/", "/empty", "/photo.jpg", "/photos", "/photos/c.jpg"})
		})
	}

	t.Run("file", func(t *testing.T) {
		storage := seed(t, factory)

		if err := storage.RemoveAll(ctx, "/photo.jpg"); err != nil {
			t.Fatal(err)
		}

		_, err := storage.Stat(ctx, "/photo.jpg")
		assertNotExist(t, err)

		if _, err = storage.Stat(ctx, "/photo/a.jpg"); err != nil {
			t.Errorf("sibling directory damaged: %s", err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		storage := seed(t, factory)

		for _, name := range []string{"/missing", "/pho", "/photo.jp"} {
			if err := storage.RemoveAll(ctx, name); err != nil {
				t.Errorf("RemoveAll(%s) unexpected error: %s", name, err)
			}
		}

		got, err := walkAll(storage, "/")
		if err != nil {
			t.Fatal(err)
		}

		if len(got) != 9 {
			t.Errorf("RemoveAll() of a missing path changed the tree: %v", got)
		}
	})

	t.Run("root", func(t *testing.T) {
		storage := seed(t, factory)

		for _, name := range []string{"/", ""} {
			if err := storage.RemoveAll(ctx, name); err == nil {
				t.Errorf("RemoveAll(%q) want an error for the root", name)
			}
		}

		if _, err := storage.Stat(ctx, "/photo/a.jpg"); err != nil {
			t.Errorf("tree damaged: %s", err)
		}
	})
}
