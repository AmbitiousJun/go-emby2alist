package https

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var client *http.Client

// RedirectCodes 有重定向含义的 http 响应码
var RedirectCodes = [4]int{http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect}

func init() {
	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			// 建立连接 1 分钟超时
			Dial:                  (&net.Dialer{Timeout: time.Minute}).Dial,
			// 接收数据 5 分钟超时
			ResponseHeaderTimeout: time.Minute * 5,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ExtractReqBody 克隆并提取请求体
// 不影响 c 对象之后再次读取请求体
func ExtractReqBody(c *gin.Context) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes, nil
}

// ClientRequestHost 获取客户端请求的 Host
func ClientRequestHost(c *gin.Context) string {
	if c == nil {
		return ""
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// ClientRequestUrl 获取客户端请求的完整地址
func ClientRequestUrl(c *gin.Context) string {
	return fmt.Sprintf("%s%s", ClientRequestHost(c), c.Request.URL.String())
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

// IsErrorResponse 判断一个请求响应是否是错误响应
//
// 判断标准是响应码以 4xx 5xx 开头
func IsErrorResponse(c *gin.Context) bool {
	if c == nil {
		return true
	}
	code := c.Writer.Status()
	str := strconv.Itoa(code)
	return strings.HasPrefix(str, "4") || strings.HasPrefix(str, "5")
}

// MapBody 将 map 转换为 ReadCloser 流
func MapBody(body map[string]interface{}) io.ReadCloser {
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

// CloneHeader 克隆 http 头部到 gin 的响应头中
func CloneHeader(c *gin.Context, header http.Header) {
	if c == nil || header == nil {
		return
	}
	for key, values := range header {
		c.Writer.Header().Del(key)
		for _, value := range values {
			c.Header(key, value)
		}
	}
}

// Request 发起 http 请求获取响应
func Request(method, url string, header http.Header, body io.ReadCloser) (*http.Response, error) {
	_, resp, err := RequestRedirect(method, url, header, body, false)
	return resp, err
}

// RequestRedirect 发起 http 请求获取响应
//
// 如果一个请求有多次重定向并且进行了 autoRedirect,
// 则最后一次重定向的 url 会作为第一个参数返回
func RequestRedirect(method, url string, header http.Header, body io.ReadCloser, autoRedirect bool) (string, *http.Response, error) {
	// 1 转换请求
	var bodyBytes []byte
	if body != nil {
		var err error
		if bodyBytes, err = io.ReadAll(body); err != nil {
			return "", nil, fmt.Errorf("读取请求体失败: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header = header

	// 2 发出请求
	resp, err := client.Do(req)
	if err != nil {
		return url, resp, err
	}

	// 3 对重定向响应的处理
	if autoRedirect && IsRedirectCode(resp.StatusCode) {
		loc := resp.Header.Get("Location")
		if strings.HasPrefix(loc, "/") {
			// 需要拼接上当前请求的前缀后再进行重定向
			loc = fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, loc)
		}
		return RequestRedirect(method, loc, header, body, autoRedirect)
	}
	return url, resp, err
}

// ProxyRequest 代理请求
func ProxyRequest(c *gin.Context, remote string, withUri bool) error {
	if c == nil || remote == "" {
		return errors.New("参数为空")
	}

	if withUri {
		remote = remote + c.Request.URL.String()
	}

	// 1 解析远程地址
	rmtUrl, err := url.Parse(remote)
	if err != nil {
		return fmt.Errorf("解析远程地址失败: %v", err)
	}

	// 2 拷贝 query 参数
	rmtUrl.RawQuery = c.Request.URL.RawQuery

	// 3 创建请求
	var bodyBuffer io.Reader = nil
	if c.Request.Body != nil {
		reqBodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return fmt.Errorf("读取请求体失败: %v", err)
		}
		if len(reqBodyBytes) > 0 {
			bodyBuffer = bytes.NewBuffer(reqBodyBytes)
		}
	}

	req, err := http.NewRequest(c.Request.Method, rmtUrl.String(), bodyBuffer)
	if err != nil {
		return fmt.Errorf("初始化请求失败: %v", err)
	}

	// 4 拷贝请求头
	req.Header = c.Request.Header

	// 5 发起请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 6 回写响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 7 回写响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %v", err)
	}
	c.Status(resp.StatusCode)
	c.Writer.Write(bodyBytes)
	c.Writer.Flush()
	return nil
}
