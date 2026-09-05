package utils

import "testing"

func TestParseS3ObjectRef(t *testing.T) {
	tests := []struct {
		raw      string
		fallback string
		bucket   string
		key      string
		ok       bool
	}{
		{
			raw:    "https://vsproofs.s3.ap-south-1.amazonaws.com/uploads/123.jpg",
			bucket: "vsproofs",
			key:    "uploads/123.jpg",
			ok:     true,
		},
		{
			raw:    "https://vsproofs.s3.amazonaws.com/uploads/a.png",
			bucket: "vsproofs",
			key:    "uploads/a.png",
			ok:     true,
		},
		{
			raw:    "https://s3.ap-south-1.amazonaws.com/vsproofs/uploads/b.jpg",
			bucket: "vsproofs",
			key:    "uploads/b.jpg",
			ok:     true,
		},
		{
			raw:      "uploads/local.jpg",
			fallback: "vsproofs",
			bucket:   "vsproofs",
			key:      "uploads/local.jpg",
			ok:       true,
		},
		{
			raw: "https://ik.imagekit.io/x/file.jpg",
			ok:  false,
		},
		{
			raw: "",
			ok:  false,
		},
	}

	for _, tt := range tests {
		bucket, key, ok := ParseS3ObjectRef(tt.raw, tt.fallback)
		if ok != tt.ok || bucket != tt.bucket || key != tt.key {
			t.Fatalf("raw=%q: got (%q, %q, %v) want (%q, %q, %v)",
				tt.raw, bucket, key, ok, tt.bucket, tt.key, tt.ok)
		}
	}
}
