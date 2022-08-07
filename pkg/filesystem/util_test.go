package filesystem

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
)

func TestCheckPathname(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := map[string]struct {
		args args
		want error
	}{
		"valid": {
			args{
				pathname: "/test",
			},
			nil,
		},
		"invalid": {
			args{
				pathname: "/test/../root",
			},
			ErrRelativePath,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := checkPathname(tc.args.pathname); got != tc.want {
				t.Errorf("checkPathname() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestPath(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := map[string]struct {
		instance App
		args     args
		want     string
	}{
		"simple": {
			App{
				rootDirectory: "/home/users",
			},
			args{
				pathname: "/test",
			},
			"/home/users/test",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := tc.instance.Path(tc.args.pathname); got != tc.want {
				t.Errorf("Path() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestGetRelativePath(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := map[string]struct {
		instance App
		args     args
		want     string
	}{
		"simple": {
			App{
				rootDirectory: "/home/users",
			},
			args{
				pathname: "/home/users/test",
			},
			"/test",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := tc.instance.getRelativePath(tc.args.pathname); got != tc.want {
				t.Errorf("getRelativePath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestGetMode(t *testing.T) {
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
			0o700,
		},
		"file": {
			args{
				name: "/photo.png",
			},
			0o600,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := getMode(tc.args.name); got != tc.want {
				t.Errorf("getMode() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestConvertToItem(t *testing.T) {
	type args struct {
		pathname string
		info     os.FileInfo
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
				ID:        "f586eea2876d83b41022dafcc2e615003dfdcce3",
				Name:      "README.md",
				Pathname:  "/README.md",
				Extension: ".md",
				IsDir:     false,
				Date:      readmeInfo.ModTime(),
				Size:      readmeInfo.Size(),
			},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := convertToItem(tc.args.pathname, tc.args.info); got != tc.want {
				t.Errorf("convertToItem() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestConvertError(t *testing.T) {
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
		"standard": {
			args{
				err: errors.New("read"),
			},
			errors.New("read"),
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			failed := false
			got := App{}.ConvertError(tc.args.err)

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

func BenchmarkConverToItem(b *testing.B) {
	info, err := os.Stat("util_test.go")
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		convertToItem("/pkg/filesystem/utils_test.go", info)
	}
}
