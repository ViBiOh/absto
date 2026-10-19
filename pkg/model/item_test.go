package model

import "testing"

func TestItemDir(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		item Item
		want string
	}{
		"directory": {
			Item{Pathname: "/photos", IsDirValue: true},
			"/photos/",
		},
		"directory with slash": {
			Item{Pathname: "/photos/", IsDirValue: true},
			"/photos/",
		},
		"root": {
			Item{Pathname: "/", IsDirValue: true},
			"/",
		},
		"file": {
			Item{Pathname: "/photos/a.jpg"},
			"/photos/",
		},
		"file at root": {
			Item{Pathname: "/a.jpg"},
			"/",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := tc.item.Dir(); got != tc.want {
				t.Errorf("Dir() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
