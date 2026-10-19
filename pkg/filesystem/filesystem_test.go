package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
)

func TestWalk(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	root := filepath.Join(base, "root")

	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(base, "outside"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(base, "outside", "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		name    string
		want    []string
		wantErr error
	}{
		"directory": {
			"/sub",
			[]string{"/sub", "/sub/a.txt"},
			nil,
		},
		"relative path": {
			"/../outside",
			nil,
			model.ErrRelativePath,
		},
		"relative path in the middle": {
			"/sub/../../outside",
			nil,
			model.ErrRelativePath,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			var got []string

			err := service.Walk(context.Background(), tc.name, func(item model.Item) error {
				got = append(got, item.Pathname)

				return nil
			})

			if err != tc.wantErr {
				t.Fatalf("Walk() error = %v, want %v", err, tc.wantErr)
			}

			sort.Strings(got)

			if !slices.Equal(got, tc.want) {
				t.Errorf("Walk() = %v, want %v", got, tc.want)
			}
		})
	}
}

func newLinkedRoot(t *testing.T) (string, string, Service) {
	t.Helper()

	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")

	for _, dir := range []string{filepath.Join(root, "sub"), outside} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	links := map[string]string{
		"relative": "../outside",
		"absolute": outside,
		"alias":    "sub",
	}

	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}

	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	return root, outside, service
}

func TestSymlinkEscape(t *testing.T) {
	t.Parallel()

	_, outside, service := newLinkedRoot(t)
	ctx := context.Background()

	operations := map[string]func(link string) error{
		"stat": func(link string) error {
			_, err := service.Stat(ctx, "/"+link+"/secret.txt")

			return err
		},
		"list": func(link string) error {
			_, err := service.List(ctx, "/"+link)

			return err
		},
		"read": func(link string) error {
			reader, err := service.ReadFrom(ctx, "/"+link+"/secret.txt")
			if err == nil {
				_ = reader.Close()
			}

			return err
		},
		"write": func(link string) error {
			return service.WriteTo(ctx, "/"+link+"/created.txt", strings.NewReader("pwned"), model.WriteOpts{})
		},
		"overwrite": func(link string) error {
			return service.WriteTo(ctx, "/"+link+"/secret.txt", strings.NewReader("pwned"), model.WriteOpts{})
		},
		"mkdir": func(link string) error {
			return service.Mkdir(ctx, "/"+link+"/created", model.DirectoryPerm)
		},
		"remove file": func(link string) error {
			return service.RemoveAll(ctx, "/"+link+"/secret.txt")
		},
		"rename from outside": func(link string) error {
			return service.Rename(ctx, "/"+link+"/secret.txt", "/stolen.txt")
		},
		"rename to outside": func(link string) error {
			return service.Rename(ctx, "/sub/a.txt", "/"+link+"/a.txt")
		},
		"update date": func(link string) error {
			return service.UpdateDate(ctx, "/"+link+"/secret.txt", time.Unix(0, 0))
		},
		"walk": func(link string) error {
			return service.Walk(ctx, "/"+link, func(model.Item) error { return nil })
		},
	}

	t.Run("operations", func(t *testing.T) {
		for intention, operation := range operations {
			for _, link := range []string{"relative", "absolute"} {
				t.Run(intention+" through "+link, func(t *testing.T) {
					t.Parallel()

					if err := operation(link); err == nil {
						t.Error("want an error when going through a symlink escaping the root")
					}
				})
			}
		}
	})

	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 || entries[0].Name() != "secret.txt" {
		t.Errorf("outside directory content changed: %v", entries)
	}

	content, err := os.ReadFile(filepath.Join(outside, "secret.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "secret" {
		t.Errorf("outside file content = %q, want `secret`", content)
	}

	if _, err := os.Stat(filepath.Join(filepath.Dir(outside), "root", "stolen.txt")); err == nil {
		t.Error("file outside the root has been moved inside")
	}
}

func TestSymlinkInsideRoot(t *testing.T) {
	t.Parallel()

	_, _, service := newLinkedRoot(t)

	reader, err := service.ReadFrom(context.Background(), "/alias/a.txt")
	if err != nil {
		t.Fatalf("ReadFrom() unexpected error: %s", err)
	}

	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "inside" {
		t.Errorf("ReadFrom() = %q, want `inside`", content)
	}
}

func TestItems(t *testing.T) {
	t.Parallel()

	root, _, _ := newLinkedRoot(t)

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	relativeRoot, err := filepath.Rel(workdir, root)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"absolute":        root,
		"absolute slash":  root + "/",
		"relative":        relativeRoot,
		"relative slash":  relativeRoot + "/",
		"relative prefix": "./" + filepath.Join(".", relativeRoot),
	}

	for intention, directory := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			service, err := New(directory)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()

			rootItem, err := service.Stat(ctx, "/")
			if err != nil {
				t.Fatalf("Stat() unexpected error: %s", err)
			}

			if rootItem.Pathname != "/" || rootItem.Name() != "root" || !rootItem.IsDir() {
				t.Errorf("Stat(/) = %+v, want root directory item", rootItem)
			}

			dirItem, err := service.Stat(ctx, "/sub/")
			if err != nil {
				t.Fatalf("Stat() unexpected error: %s", err)
			}

			if dirItem.Pathname != "/sub" || dirItem.Name() != "sub" {
				t.Errorf("Stat(/sub/) = %+v, want pathname `/sub`", dirItem)
			}

			items, err := service.List(ctx, "/sub")
			if err != nil {
				t.Fatalf("List() unexpected error: %s", err)
			}

			if len(items) != 1 || items[0].Pathname != "/sub/a.txt" || items[0].Name() != "a.txt" {
				t.Errorf("List(/sub) = %+v, want a single `/sub/a.txt`", items)
			}

			var walked []string

			err = service.Walk(ctx, "/sub", func(item model.Item) error {
				walked = append(walked, item.Pathname)

				return nil
			})
			if err != nil {
				t.Fatalf("Walk() unexpected error: %s", err)
			}

			if want := []string{"/sub", "/sub/a.txt"}; !slices.Equal(walked, want) {
				t.Errorf("Walk(/sub) = %v, want %v", walked, want)
			}
		})
	}
}

func TestRemoveSymlink(t *testing.T) {
	t.Parallel()

	root, outside, service := newLinkedRoot(t)

	if err := service.RemoveAll(context.Background(), "/relative"); err != nil {
		t.Fatalf("RemoveAll() unexpected error: %s", err)
	}

	if _, err := os.Lstat(filepath.Join(root, "relative")); err == nil {
		t.Error("symlink still exists")
	}

	if _, err := os.Stat(filepath.Join(outside, "secret.txt")); err != nil {
		t.Errorf("symlink target has been removed: %s", err)
	}
}

type readerOnly struct {
	io.Reader
}

func BenchmarkWriteTo(b *testing.B) {
	service, err := New(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	payload := strings.Repeat("a", 1<<20)

	b.ReportAllocs()

	for b.Loop() {
		if err := service.WriteTo(ctx, "/file.txt", readerOnly{strings.NewReader(payload)}, model.WriteOpts{}); err != nil {
			b.Fatal(err)
		}
	}
}
