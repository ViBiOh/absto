package model

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"
)

func TestValidPath(t *testing.T) {
	t.Parallel()

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
		"begin of string": {
			args{
				pathname: "../root",
			},
			ErrRelativePath,
		},
		"end of string": {
			args{
				pathname: "root/..",
			},
			ErrRelativePath,
		},
		"valid filename": {
			args{
				pathname: "/content/legen..dary!",
			},
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := ValidPath(tc.args.pathname); got != tc.want {
				t.Errorf("ValidPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestErrNotExist(t *testing.T) {
	t.Parallel()

	original := &fs.PathError{Op: "open", Path: "/file", Err: syscall.ENOENT}

	converted := ErrNotExist(original)

	cases := map[string]struct {
		got  bool
		want bool
	}{
		"is not exist":       {IsNotExist(converted), true},
		"is fs not exist":    {errors.Is(converted, fs.ErrNotExist), true},
		"keeps the original": {errors.Is(converted, syscall.ENOENT), true},
		"wrapped":            {IsNotExist(fmt.Errorf("walk: %w", converted)), true},
		"idempotent":         {ErrNotExist(converted) == converted, true},
		"unrelated":          {IsNotExist(errors.New("boom")), false},
		"nil":                {IsNotExist(nil), false},
		"nil input":          {IsNotExist(ErrNotExist(nil)), true},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Errorf("got %t, want %t", tc.got, tc.want)
			}
		})
	}

	if got, want := converted.Error(), original.Error()+": not exists"; got != want {
		t.Errorf("Error() = `%s`, want `%s`", got, want)
	}
}
