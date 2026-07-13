package testutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

func SignHMACSHA512(secret []byte, body []byte) string {
	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func SignHMACSHA256(secret []byte, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// HashSHA256 returns the hex-encoded SHA-256 digest of data.
// Flutterwave's Verif-Hash is SHA256(secretHash), NOT an HMAC.
func HashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
