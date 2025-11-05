package helpers

import (
	"os"
	"testing"

	"github.com/elskow/go-microservice-template/config"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		envCost     string
		expectedMin int
		wantErr     bool
	}{
		{
			name:        "hash password with custom cost 10",
			password:    "testpassword123",
			envCost:     "10",
			expectedMin: 10,
			wantErr:     false,
		},
		{
			name:        "hash password with cost below minimum (enforced to 10)",
			password:    "testpassword123",
			envCost:     "4",
			expectedMin: 10,
			wantErr:     false,
		},
		{
			name:        "hash password with cost 11",
			password:    "testpassword123",
			envCost:     "11",
			expectedMin: 11,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("BCRYPT_COST", tt.envCost)
			defer os.Unsetenv("BCRYPT_COST")
			defer config.Reset()
			// Reload config to pick up new env var
			config.Load()

			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}

				if hash == tt.password {
					t.Error("HashPassword() returned unhashed password")
				}

				cost, err := bcrypt.Cost([]byte(hash))
				if err != nil {
					t.Errorf("Failed to get bcrypt cost: %v", err)
				}

				if cost < tt.expectedMin {
					t.Errorf("BCrypt cost = %d, expected minimum %d", cost, tt.expectedMin)
				}

				if !CheckPassword(tt.password, hash) {
					t.Error("CheckPassword() failed to verify correct password")
				}
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	os.Setenv("BCRYPT_COST", "10")
	defer os.Unsetenv("BCRYPT_COST")
	defer config.Reset()
	config.Load()

	password := "testpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		plain    string
		hash     string
		expected bool
	}{
		{
			name:     "correct password",
			plain:    password,
			hash:     hash,
			expected: true,
		},
		{
			name:     "incorrect password",
			plain:    "wrongpassword",
			hash:     hash,
			expected: false,
		},
		{
			name:     "empty password",
			plain:    "",
			hash:     hash,
			expected: false,
		},
		{
			name:     "empty hash",
			plain:    password,
			hash:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPassword(tt.plain, tt.hash)
			if result != tt.expected {
				t.Errorf("CheckPassword() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetBcryptCost(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "default cost when no env set",
			envValue: "",
			expected: 12,
		},
		{
			name:     "valid cost within range",
			envValue: "11",
			expected: 11,
		},
		{
			name:     "cost below minimum",
			envValue: "5",
			expected: 10,
		},
		{
			name:     "cost at minimum",
			envValue: "10",
			expected: 10,
		},
		{
			name:     "cost above maximum",
			envValue: "50",
			expected: 31,
		},
		{
			name:     "cost 12 (production default)",
			envValue: "12",
			expected: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer config.Reset()
			if tt.envValue != "" {
				os.Setenv("BCRYPT_COST", tt.envValue)
				defer os.Unsetenv("BCRYPT_COST")
			} else {
				os.Unsetenv("BCRYPT_COST")
			}

			cfg := config.Load()
			result := cfg.BcryptCost
			if result != tt.expected {
				t.Errorf("getBcryptCost() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func BenchmarkHashPassword(b *testing.B) {
	os.Setenv("BCRYPT_COST", "10")
	defer os.Unsetenv("BCRYPT_COST")
	defer config.Reset()
	config.Load()

	password := "testpassword123"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	os.Setenv("BCRYPT_COST", "10")
	defer os.Unsetenv("BCRYPT_COST")
	defer config.Reset()
	config.Load()

	password := "testpassword123"
	hash, _ := HashPassword(password)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CheckPassword(password, hash)
	}
}
