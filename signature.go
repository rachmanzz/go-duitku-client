package duitku

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func generateSignature(merchantCode, timestamp, apiKey string) string {
	stringToSign := merchantCode + timestamp
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyCallbackSignature(merchantCode string, amount int64, merchantOrderID, apiKey, signature string) bool {
	stringToSign := fmt.Sprintf("%s%d%s", merchantCode, amount, merchantOrderID)
	expected := hmacSHA256(stringToSign, apiKey)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func hmacSHA256(stringToSign, apiKey string) string {
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil))
}
