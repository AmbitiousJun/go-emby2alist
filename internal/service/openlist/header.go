package openlist

import "net/http"

var openlistHeaderKeys = []string{"User-Agent"}

// CleanHeader 清理请求头
func CleanHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}

	newHeader := make(http.Header)
	for _, key := range openlistHeaderKeys {
		if value := header.Get(key); value != "" {
			newHeader.Add(key, value)
		}
	}
	return newHeader
}
