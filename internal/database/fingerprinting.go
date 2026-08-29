package database

import (
	"crypto/sha256"
	"encoding/hex"
)

func fingerprintSQL(query string) string {
	sum := sha256.Sum256([]byte(query))

	return hex.EncodeToString(sum[:8])
}
