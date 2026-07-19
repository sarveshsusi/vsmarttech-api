package utils

import "testing"

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr bool
	}{
		{"too short", "Ab1", true},
		{"missing upper", "abcdefg1", true},
		{"missing lower", "ABCDEFG1", true},
		{"missing number", "Abcdefgh", true},
		{"valid", "Abcdefg1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePasswordStrength(%q) err=%v, wantErr=%v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("Abcdefg1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := CheckPassword("Abcdefg1", hash); err != nil {
		t.Fatalf("CheckPassword valid: %v", err)
	}
	if err := CheckPassword("WrongPass1", hash); err == nil {
		t.Fatal("CheckPassword expected error for wrong password")
	}
}
