package encrypts

import (
	"crypto/md5"
	"encoding/hex"
)

// Md5Hash 对字符串 raw 进行 md5 哈希运算, 返回十六进制
func Md5Hash(raw string) string {
	hash := md5.New()
	hash.Write([]byte(raw))
	return hex.EncodeToString(hash.Sum(nil))
}
