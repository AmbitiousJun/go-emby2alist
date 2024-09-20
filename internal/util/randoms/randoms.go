package randoms

import (
	"math/rand"
	"strings"
)

// hexs 16 进制字符
var hexs = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}

// RandomHex 随机返回一串 16 进制的字符串, 可通过 n 指定长度
func RandomHex(n int) string {
	if n <= 0 {
		return ""
	}
	sb := strings.Builder{}
	for n > 0 {
		idx := rand.Intn(len(hexs))
		sb.WriteString(hexs[idx])
		n--
	}
	return sb.String()
}
