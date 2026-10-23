package s3

import (
	"net/http"
	"testing"
)

func TestCopyDestination(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		storageClass  string
		wantClass     string
		wantDirective string
	}{
		"no storage class": {
			"",
			"",
			"",
		},
		"storage class": {
			"STANDARD_IA",
			"STANDARD_IA",
			"REPLACE",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			service := Service{bucket: "bucket", storageClass: tc.storageClass}

			options := service.copyDestination("dir/file.txt")
			if options.Bucket != "bucket" || options.Object != "dir/file.txt" {
				t.Errorf("copyDestination() = %s/%s, want bucket/dir/file.txt", options.Bucket, options.Object)
			}

			header := make(http.Header)
			options.Marshal(header)

			if got := header.Get("X-Amz-Storage-Class"); got != tc.wantClass {
				t.Errorf("storage class header = `%s`, want `%s`", got, tc.wantClass)
			}

			if got := header.Get("X-Amz-Metadata-Directive"); got != tc.wantDirective {
				t.Errorf("metadata directive header = `%s`, want `%s`", got, tc.wantDirective)
			}
		})
	}
}
