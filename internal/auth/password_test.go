package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt позволяет пустые пароли
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 100),
			wantErr:  false,
		},
		{
			name:     "special characters",
			password: "p@ssw0rd!#$%",
			wantErr:  false,
		},
		{
			name:     "unicode password",
			password: "пароль123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Проверяем, что хеш не пустой
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				// Проверяем, что хеш начинается с bcrypt префикса
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
					t.Errorf("HashPassword() hash doesn't look like bcrypt: %s", hash)
				}
				// Проверяем, что хеш отличается от исходного пароля
				if hash == tt.password {
					t.Error("HashPassword() returned password as hash")
				}
			}
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "test123"

	// Генерируем два хеша одного пароля
	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Хеши должны быть разными (bcrypt использует соль)
	if hash1 == hash2 {
		t.Error("HashPassword() produced identical hashes for same password")
	}

	// Но оба должны проходить проверку
	if !CheckPassword(password, hash1) {
		t.Error("CheckPassword() failed for hash1")
	}
	if !CheckPassword(password, hash2) {
		t.Error("CheckPassword() failed for hash2")
	}
}

func TestCheckPassword(t *testing.T) {
	correctPassword := "correct123"
	hash, err := HashPassword(correctPassword)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: correctPassword,
			hash:     hash,
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrong123",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "similar password",
			password: "correct124",
			hash:     hash,
			want:     false,
		},
		{
			name:     "case sensitive",
			password: "Correct123",
			hash:     hash,
			want:     false,
		},
		{
			name:     "invalid hash",
			password: correctPassword,
			hash:     "invalid-hash",
			want:     false,
		},
		{
			name:     "empty hash",
			password: correctPassword,
			hash:     "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPasswordEdgeCases(t *testing.T) {
	t.Run("empty password empty hash", func(t *testing.T) {
		if CheckPassword("", "") {
			t.Error("CheckPassword() should return false for empty password and hash")
		}
	})

	t.Run("long password", func(t *testing.T) {
		longPassword := strings.Repeat("a", 100)
		hash, err := HashPassword(longPassword)
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}
		if !CheckPassword(longPassword, hash) {
			t.Error("CheckPassword() failed for long password")
		}
	})

	t.Run("unicode password", func(t *testing.T) {
		unicodePassword := "пароль_密码_🔐"
		hash, err := HashPassword(unicodePassword)
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}
		if !CheckPassword(unicodePassword, hash) {
			t.Error("CheckPassword() failed for unicode password")
		}
	})
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmark123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	password := "benchmark123"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckPassword(password, hash)
	}
}
