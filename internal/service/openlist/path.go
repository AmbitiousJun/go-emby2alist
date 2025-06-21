package openlist

import "encoding/base64"

// PathEncode 将 alist 的资源原始路径进行编码, 防止路径在传输过程中出现错误
func PathEncode(rawPath string) string {
	return base64.StdEncoding.EncodeToString([]byte(rawPath))
}

// PathDecode 对 alist 的编码路径进行解码, 返回原始路径
//
// 如果解码失败, 则返回原路径
func PathDecode(encPath string) string {
	res, err := base64.StdEncoding.DecodeString(encPath)
	if err != nil {
		return encPath
	}
	return string(res)
}
