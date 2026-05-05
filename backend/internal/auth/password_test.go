package auth

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestVerifyPassword(t *testing.T) {
	okHash, err := bcrypt.GenerateFromPassword([]byte("secret-123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() err = %v", err)
	}

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password",
			hash:     string(okHash),
			password: "secret-123",
		},
		{
			name:     "invalid password",
			hash:     string(okHash),
			password: "wrong-password",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "invalid hash format",
			hash:     "not-bcrypt-hash",
			password: "secret-123",
			wantErr:  bcrypt.ErrHashTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPassword(tt.hash, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("VerifyPassword() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyPassword() err = %v", err)
			}
		})
	}
}
