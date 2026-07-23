package utils

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDetectImageContentType(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	ct, err := DetectImageContentType(buf.Bytes())
	if err != nil || ct != "image/png" {
		t.Fatalf("got %q err=%v", ct, err)
	}
	if err := ValidateDecodableImage(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestDetectImageContentTypeRejectsHTML(t *testing.T) {
	payload := []byte("<html><script>alert(1)</script></html>")
	if _, err := DetectImageContentType(payload); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestSafeUploadFilename(t *testing.T) {
	if SafeUploadFilename("image/jpeg") != ".jpg" {
		t.Fatal("jpeg ext")
	}
	if SanitizeOriginalFilename("../../etc/passwd") != "passwd" {
		t.Fatal("path traversal stripped")
	}
}
