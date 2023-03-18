package model

import "testing"

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
			if got := CheckRelativePath(tc.args.pathname); got != tc.want {
				t.Errorf("checkPathname() = %t, want %t", got, tc.want)
			}
		})
	}
}
