package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "password123" {
		t.Fatal("hash が平文のまま")
	}
	if err := VerifyPassword(hash, "password123"); err != nil {
		t.Fatalf("VerifyPassword (一致): %v", err)
	}
	if err := VerifyPassword(hash, "wrong"); err == nil {
		t.Fatal("VerifyPassword (不一致) が nil")
	}
	if err := VerifyPassword("", "password123"); err == nil {
		t.Fatal("空ハッシュは常に不一致であるべき")
	}
}

func TestNewSessionToken(t *testing.T) {
	token, id, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	if token == "" || id == "" {
		t.Fatal("token / id が空")
	}
	if token == id {
		t.Fatal("生トークンと DB 格納 id が同一 (ハッシュされていない)")
	}
	if HashToken(token) != id {
		t.Fatal("HashToken(token) が id と一致しない")
	}

	token2, _, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	if token == token2 {
		t.Fatal("トークンが衝突 (乱数になっていない)")
	}
}
