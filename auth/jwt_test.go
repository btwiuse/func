package auth

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestClaims(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		keyProvider KeyProvider
		expected    jwt.Expected
		want        jwt.Claims
		wantErr     bool
	}{
		{
			name:    "EmptyToken",
			token:   "",
			wantErr: true,
		},
		{
			name:    "InvalidToken",
			token:   "this.is.not.a.jwt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Claims(tt.token, tt.keyProvider, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Claims() err = %v, want err = %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Diff (-got +want)\n%s", diff)
			}
		})
	}
}
