// Package auth は暗号方式 (bcrypt / SHA-256) の選択を 1 箇所に閉じ、handler/repository から
// その詳細を切り離すために置く。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// 生トークンは Cookie だけが持ち、DB には SHA-256 ハッシュを格納する。
// DB 流出時に生トークンを復元できないようにするため。
func NewSessionToken() (token, id string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b[:])
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
