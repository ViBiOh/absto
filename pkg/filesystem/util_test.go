package filesystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
)

func TestPath(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}

	cases := map[string]struct {
		instance Service
		args     args
		want     string
	}{
		"simple": {
			Service{
				rootDirectory: "/home/users",
			},
			args{
				name: "/test",
			},
			"/home/users/test",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()
			if got := tc.instance.Path(tc.args.name); got != tc.want {
				t.Errorf("Path() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestGetMode(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}

	cases := map[string]struct {
		args args
		want os.FileMode
	}{
		"directory": {
			args{
				name: "/photos/",
			},
			model.DirectoryPerm,
		},
		"file": {
			args{
				name: "/photo.png",
			},
			model.RegularFilePerm,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := getMode(tc.args.name); got != tc.want {
				t.Errorf("getMode() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestConvertToItem(t *testing.T) {
	t.Parallel()

	type args struct {
		pathname string
		info     fs.FileInfo
	}

	readmeInfo, err := os.Stat("../../README.md")
	if err != nil {
		t.Error(err)
	}

	cases := map[string]struct {
		args args
		want model.Item
	}{
		"simple": {
			args{
				pathname: "/README.md",
				info:     readmeInfo,
			},
			model.Item{
				ID:         "8490ed15d311ea4c",
				NameValue:  "README.md",
				Pathname:   "/README.md",
				Extension:  ".md",
				IsDirValue: false,
				Date:       readmeInfo.ModTime(),
				SizeValue:  readmeInfo.Size(),
				FileMode:   readmeInfo.Mode(),
			},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := convertToItem(tc.args.pathname, tc.args.info); got != tc.want {
				t.Errorf("convertToItem() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestConvertError(t *testing.T) {
	t.Parallel()

	type args struct {
		err error
	}

	cases := map[string]struct {
		args args
		want error
	}{
		"nil": {
			args{
				err: nil,
			},
			nil,
		},
		"not exist": {
			args{
				err: os.ErrNotExist,
			},
			model.ErrNotExist(os.ErrNotExist),
		},
		"not a directory": {
			args{
				err: &fs.PathError{Op: "readdirent", Path: "file", Err: syscall.ENOTDIR},
			},
			model.ErrNotExist(errors.New("readdirent file: not a directory")),
		},
		"wrapped not a directory": {
			args{
				err: fmt.Errorf("walk: %w", &fs.PathError{Op: "readdirent", Path: "file", Err: syscall.ENOTDIR}),
			},
			model.ErrNotExist(errors.New("walk: readdirent file: not a directory")),
		},
		"standard": {
			args{
				err: errors.New("read"),
			},
			errors.New("read"),
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			failed := false
			got := Service{}.ConvertError(tc.args.err)

			if tc.want == nil && got != nil {
				failed = true
			} else if tc.want != nil && got == nil {
				failed = true
			} else if tc.want != nil && !strings.Contains(got.Error(), tc.want.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("convertError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func BenchmarkConvertToItem(b *testing.B) {
	info, err := os.Stat("util_test.go")
	if err != nil {
		b.Error(err)
	}

	for b.Loop() {
		convertToItem("/pkg/filesystem/utils_test.go", info)
	}
}

func BenchmarkJsonItem(b *testing.B) {
	info, err := os.Stat("util_test.go")
	if err != nil {
		b.Error(err)
	}

	item := convertToItem("/pkg/filesystem/utils_test.go", info)

	for b.Loop() {
		_, _ = json.Marshal(item)
	}
}
