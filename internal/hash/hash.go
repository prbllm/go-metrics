package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func ComputeHash(key string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(key))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
