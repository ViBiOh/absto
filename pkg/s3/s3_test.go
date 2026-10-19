package s3

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
)

const (
	testBucket = "bucket"
	testDate   = "2024-01-01T00:00:00.000Z"
)

type fakeServer struct {
	objects map[string]string
	mutex   sync.Mutex
	listErr bool
}

type listEntry struct {
	key      string
	isPrefix bool
}

func (fake *fakeServer) keys() []string {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	keys := make([]string, 0, len(fake.objects))
	for key := range fake.objects {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func (fake *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	bucket, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if bucket != testBucket {
		http.Error(w, "unknown bucket", http.StatusNotFound)

		return
	}

	switch {
	case key == "" && r.Method == http.MethodGet:
		fake.list(w, r)
	case r.Method == http.MethodHead:
		fake.head(w, key)
	case r.Method == http.MethodDelete:
		delete(fake.objects, key)
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodPut:
		fake.put(w, r, key)
	default:
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}
}

func (fake *fakeServer) head(w http.ResponseWriter, key string) {
	content, ok := fake.objects[key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
	w.Header().Set("ETag", `"abc"`)
	w.WriteHeader(http.StatusOK)
}

func (fake *fakeServer) put(w http.ResponseWriter, r *http.Request, key string) {
	source := r.Header.Get("X-Amz-Copy-Source")
	if source == "" {
		fake.objects[key] = ""
		w.WriteHeader(http.StatusOK)

		return
	}

	unescaped, err := url.PathUnescape(source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	_, sourceKey, _ := strings.Cut(strings.TrimPrefix(unescaped, "/"), "/")

	content, ok := fake.objects[sourceKey]
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	fake.objects[key] = content

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(`<CopyObjectResult><LastModified>` + testDate + `</LastModified><ETag>"abc"</ETag></CopyObjectResult>`))
}

func (fake *fakeServer) list(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	if fake.listErr {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))

		return
	}

	query := r.URL.Query()
	prefix := query.Get("prefix")
	delimiter := query.Get("delimiter")

	maxKeys := 1000
	if value, err := strconv.Atoi(query.Get("max-keys")); err == nil && value > 0 {
		maxKeys = value
	}

	var start int
	if token := query.Get("continuation-token"); token != "" {
		start, _ = strconv.Atoi(token)
	}

	var entries []listEntry
	seenPrefixes := make(map[string]struct{})

	for key := range fake.objects {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		rest := strings.TrimPrefix(key, prefix)

		if delimiter != "" {
			if index := strings.Index(rest, delimiter); index >= 0 {
				commonPrefix := prefix + rest[:index+len(delimiter)]
				if _, seen := seenPrefixes[commonPrefix]; !seen {
					seenPrefixes[commonPrefix] = struct{}{}
					entries = append(entries, listEntry{key: commonPrefix, isPrefix: true})
				}

				continue
			}
		}

		entries = append(entries, listEntry{key: key})
	}

	slices.SortFunc(entries, func(a, b listEntry) int { return strings.Compare(a.key, b.key) })

	end := min(start+maxKeys, len(entries))
	truncated := end < len(entries)

	var output strings.Builder

	output.WriteString(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>` + testBucket + `</Name>`)
	output.WriteString(`<KeyCount>` + strconv.Itoa(end-start) + `</KeyCount><MaxKeys>` + strconv.Itoa(maxKeys) + `</MaxKeys>`)
	output.WriteString(`<IsTruncated>` + strconv.FormatBool(truncated) + `</IsTruncated>`)

	if truncated {
		output.WriteString(`<NextContinuationToken>` + strconv.Itoa(end) + `</NextContinuationToken>`)
	}

	for _, entry := range entries[start:end] {
		var escaped strings.Builder

		_ = xml.EscapeText(&escaped, []byte(entry.key))

		if entry.isPrefix {
			output.WriteString(`<CommonPrefixes><Prefix>` + escaped.String() + `</Prefix></CommonPrefixes>`)

			continue
		}

		output.WriteString(`<Contents><Key>` + escaped.String() + `</Key><LastModified>` + testDate + `</LastModified><ETag>"abc"</ETag><Size>` + strconv.Itoa(len(fake.objects[entry.key])) + `</Size></Contents>`)
	}

	output.WriteString(`</ListBucketResult>`)

	_, _ = w.Write([]byte(output.String()))
}

func newTestService(t *testing.T, listErr bool, keys ...string) (Service, *fakeServer) {
	t.Helper()

	fake := &fakeServer{
		objects: make(map[string]string, len(keys)),
		listErr: listErr,
	}

	for _, key := range keys {
		fake.objects[key] = "content"
	}

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	service, err := New(strings.TrimPrefix(server.URL, "http://"), "access", "secret", testBucket, false, 5<<20, WithRegion("us-east-1"))
	if err != nil {
		t.Fatalf("new service: %s", err)
	}

	return service, fake
}

var testKeys = []string{"photo/", "photo/a.jpg", "photos/b.jpg", "photo.jpg"}

func TestList(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		pathname string
		want     []string
	}{
		"directory with slash": {
			"/photo/",
			[]string{"/photo/a.jpg"},
		},
		"directory without slash": {
			"/photo",
			[]string{"/photo/a.jpg"},
		},
		"root": {
			"/",
			[]string{"/photo.jpg", "/photo/", "/photos/"},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			service, _ := newTestService(t, false, testKeys...)

			items, err := service.List(context.Background(), tc.pathname)
			if err != nil {
				t.Fatalf("List() unexpected error: %s", err)
			}

			got := make([]string, 0, len(items))
			for _, item := range items {
				got = append(got, item.Pathname)
			}

			sort.Strings(got)

			if !slices.Equal(got, tc.want) {
				t.Errorf("List() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestListError(t *testing.T) {
	t.Parallel()

	service, _ := newTestService(t, true, testKeys...)

	items, err := service.List(context.Background(), "/photo/")
	if err == nil {
		t.Fatalf("List() = %v, want an error", items)
	}

	if model.IsNotExist(err) {
		t.Errorf("List() error `%s` must not be a not exist error", err)
	}
}

func TestStatDirectory(t *testing.T) {
	t.Parallel()

	t.Run("exists", func(t *testing.T) {
		t.Parallel()

		service, _ := newTestService(t, false, "dir/a.txt")

		item, err := service.Stat(context.Background(), "/dir/")
		if err != nil {
			t.Fatalf("Stat() unexpected error: %s", err)
		}

		if !item.IsDir() {
			t.Errorf("Stat() = %+v, want a directory", item)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()

		service, _ := newTestService(t, false, "dir/a.txt")

		_, err := service.Stat(context.Background(), "/other/")
		if !model.IsNotExist(err) {
			t.Errorf("Stat() error = %v, want not exist", err)
		}
	})

	t.Run("listing failure", func(t *testing.T) {
		t.Parallel()

		service, _ := newTestService(t, true, "dir/a.txt")

		_, err := service.Stat(context.Background(), "/dir/")
		if err == nil {
			t.Fatal("Stat() want an error when listing fails")
		}

		if model.IsNotExist(err) {
			t.Errorf("Stat() error `%s` must not be a not exist error", err)
		}
	})
}

func TestWalk(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		pathname string
		want     []string
	}{
		"directory without slash": {
			"/photo",
			[]string{"/photo/", "/photo/a.jpg"},
		},
		"directory with slash": {
			"/photo/",
			[]string{"/photo/", "/photo/a.jpg"},
		},
		"file": {
			"/photo.jpg",
			[]string{"/photo.jpg"},
		},
		"missing": {
			"/phot",
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			service, _ := newTestService(t, false, testKeys...)

			var got []string

			err := service.Walk(context.Background(), tc.pathname, func(item model.Item) error {
				got = append(got, item.Pathname)

				return nil
			})
			if err != nil {
				t.Fatalf("Walk() unexpected error: %s", err)
			}

			sort.Strings(got)

			if !slices.Equal(got, tc.want) {
				t.Errorf("Walk() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWalkError(t *testing.T) {
	t.Parallel()

	service, _ := newTestService(t, true, testKeys...)

	var called int

	err := service.Walk(context.Background(), "/photo/", func(model.Item) error {
		called++

		return nil
	})
	if err == nil {
		t.Error("Walk() want an error when listing fails")
	}

	if called != 0 {
		t.Errorf("Walk() called the callback %d times, want 0", called)
	}
}

func TestRemoveAll(t *testing.T) {
	t.Parallel()

	for _, pathname := range []string{"/photo", "/photo/"} {
		t.Run(pathname, func(t *testing.T) {
			t.Parallel()

			service, fake := newTestService(t, false, testKeys...)

			if err := service.RemoveAll(context.Background(), pathname); err != nil {
				t.Fatalf("RemoveAll() unexpected error: %s", err)
			}

			want := []string{"photo.jpg", "photos/b.jpg"}
			if got := fake.keys(); !slices.Equal(got, want) {
				t.Errorf("RemoveAll() left %v, want %v", got, want)
			}
		})
	}
}

func TestRemoveAllError(t *testing.T) {
	t.Parallel()

	service, fake := newTestService(t, true, testKeys...)

	if err := service.RemoveAll(context.Background(), "/photo"); err == nil {
		t.Error("RemoveAll() want an error when listing fails")
	}

	if got := fake.keys(); len(got) != len(testKeys) {
		t.Errorf("RemoveAll() left %v, want everything untouched", got)
	}
}

func TestRename(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		oldName string
		newName string
	}{
		"no slash":       {"/photo", "/img"},
		"old slash":      {"/photo/", "/img"},
		"new slash":      {"/photo", "/img/"},
		"both slash":     {"/photo/", "/img/"},
		"leading slash":  {"photo", "img"},
		"only new slash": {"photo", "img/"},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			service, fake := newTestService(t, false, testKeys...)

			if err := service.Rename(context.Background(), tc.oldName, tc.newName); err != nil {
				t.Fatalf("Rename() unexpected error: %s", err)
			}

			want := []string{"img/", "img/a.jpg", "photo.jpg", "photos/b.jpg"}
			if got := fake.keys(); !slices.Equal(got, want) {
				t.Errorf("Rename() produced %v, want %v", got, want)
			}
		})
	}
}
