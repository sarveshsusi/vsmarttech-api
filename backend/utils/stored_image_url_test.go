package utils

import "testing"

func TestIsAllowedStoredImageURL(t *testing.T) {
	t.Parallel()

	ok := []string{
		"",
		"https://vsproofs.s3.ap-south-1.amazonaws.com/uploads/a.jpg",
		"http://localhost:8080/uploads/a.jpg",
		"https://crm.example.com/uploads/photo.webp",
	}
	for _, raw := range ok {
		if !IsAllowedStoredImageURL(raw) {
			t.Fatalf("expected allowed: %q", raw)
		}
	}

	blocked := []string{
		"javascript:alert(1)",
		"data:image/png;base64,abc",
		"https://evil.example/photo.jpg",
		"ftp://localhost/uploads/a.jpg",
	}
	for _, raw := range blocked {
		if IsAllowedStoredImageURL(raw) {
			t.Fatalf("expected blocked: %q", raw)
		}
	}
}
