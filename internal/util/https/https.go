package https

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (

	// MaxRedirectDepth 重定向的最大深度
	MaxRedirectDepth = 10
)

var client *http.Client

// RedirectCodes 有重定向含义的 http 响应码
var RedirectCodes = [4]int{http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect}

func init() {
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// 建立连接 1 分钟超时
			Dial: (&net.Dialer{Timeout: time.Minute}).Dial,
			// 接收数据 5 分钟超时
			ResponseHeaderTimeout: time.Minute * 5,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// IsRedirectCode 判断 http code 是否是重定向
//
// 301, 302, 307, 308
func IsRedirectCode(code int) bool {
	for _, valid := range RedirectCodes {
		if code == valid {
			return true
		}
	}
	return false
}

// IsSuccessCode 判断 http code 是否为成功状态
func IsSuccessCode(code int) bool {
	codeStr := strconv.Itoa(code)
	return strings.HasPrefix(codeStr, "2")
}

// IsErrorCode 判断 http code 是否为错误状态
func IsErrorCode(code int) bool {
	codeStr := strconv.Itoa(code)
	return strings.HasPrefix(codeStr, "4") || strings.HasPrefix(codeStr, "5")
}

// MapBody 将 map 转换为 ReadCloser 流
func MapBody(body map[string]any) io.ReadCloser {
	if body == nil {
		return nil
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		log.Printf("MapBody 转换失败, body: %v, err : %v", body, err)
		return nil
	}
	return io.NopCloser(bytes.NewBuffer(bodyBytes))
}
