package onceo

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
)

func VerifyHMACSHA512(secret []byte, body []byte, signature string) error {
	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

func ExtractHeader(headers http.Header, key string) (string, error) {
	v := headers.Get(key)
	if v == "" {
		return "", ErrMissingHeader
	}
	return v, nil
}
