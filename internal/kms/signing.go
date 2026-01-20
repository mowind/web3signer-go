package kms

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// CalculateContentSHA256 计算内容的 SHA256 哈希（base64编码）
func CalculateContentSHA256(data []byte) string {
	if len(data) == 0 {
		// 空内容的 SHA256
		hash := sha256.Sum256([]byte(""))
		return base64.StdEncoding.EncodeToString(hash[:])
	}
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// BuildSigningString 构建签名字符串（根据文档规范）
func BuildSigningString(verb, contentSHA256, contentType, date string) string {
	// 格式：VERB + "\n" + Content-SHA256 + "\n" + Content-Type + "\n" + Date
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		verb,
		contentSHA256,
		contentType,
		date,
	)
}

// CalculateHMACSHA256 计算 HMAC-SHA256 签名（base64编码）
func CalculateHMACSHA256(message, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// BuildAuthorizationHeader 构建 Authorization 头（根据文档规范）
func BuildAuthorizationHeader(accessKeyID, signature string) string {
	// 格式：MPC-KMS AK:Signature
	return fmt.Sprintf("MPC-KMS %s:%s", accessKeyID, signature)
}
