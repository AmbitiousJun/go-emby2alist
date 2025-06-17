package https

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

type RequestHolder struct {

	// method 请求方法
	method string

	// url 请求地址
	url string

	// header 请求头
	header http.Header

	// body 请求体
	body io.ReadCloser

	// redirect 是否自动重定向
	redirect bool
}

// Request 构造自定义请求
func Request(method, url string) *RequestHolder {
	return &RequestHolder{method: method, url: url}
}

// Get 构造 GET 请求
func Get(url string) *RequestHolder {
	return &RequestHolder{method: http.MethodGet, url: url}
}

// Post 构造 POST 请求
func Post(url string) *RequestHolder {
	return &RequestHolder{method: http.MethodPost, url: url}
}

// Delete 构造 DELETE 请求
func Delete(url string) *RequestHolder {
	return &RequestHolder{method: http.MethodDelete, url: url}
}

// Put 构造 PUT 请求
func Put(url string) *RequestHolder {
	return &RequestHolder{method: http.MethodPut, url: url}
}

// Options 构造 OPTIONS 请求
func Options(url string) *RequestHolder {
	return &RequestHolder{method: http.MethodOptions, url: url}
}

// Head 构造 HEAD 请求
func Head(url string) *RequestHolder {
	return &RequestHolder{method: http.MethodHead, url: url}
}

// AddHeader 添加请求头字段
func (r *RequestHolder) AddHeader(key, value string) *RequestHolder {
	if r.header == nil {
		r.header = make(http.Header)
	}
	r.header.Add(key, value)
	return r
}

// Header 设置请求头
func (r *RequestHolder) Header(header http.Header) *RequestHolder {
	r.header = header
	return r
}

// Body 设置请求体
func (r *RequestHolder) Body(body io.ReadCloser) *RequestHolder {
	r.body = body
	return r
}

// Do 发起请求 自动重定向
func (r *RequestHolder) Do() (*http.Response, error) {
	r.redirect = true
	_, resp, err := r.execute()
	return resp, err
}

// DoSingle 发起请求 不自动重定向
func (r *RequestHolder) DoSingle() (*http.Response, error) {
	_, resp, err := r.execute()
	return resp, err
}

// DoRedirect 发起请求 自动重定向 获取最终地址
func (r *RequestHolder) DoRedirect() (string, *http.Response, error) {
	r.redirect = true
	return r.execute()
}

// execute 发起 http 请求获取响应
//
// 如果一个请求有多次重定向并且进行了 autoRedirect,
// 则最后一次重定向的 url 会作为第一个参数返回
func (r *RequestHolder) execute() (string, *http.Response, error) {
	var inner func(method, url string, header http.Header, body io.ReadCloser, autoRedirect bool, depth int) (string, *http.Response, error)
	inner = func(method, url string, header http.Header, body io.ReadCloser, autoRedirect bool, depth int) (string, *http.Response, error) {
		if depth >= MaxRedirectDepth {
			return url, nil, fmt.Errorf("重定向次数过多: %s", url)
		}

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
		if !autoRedirect || !IsRedirectCode(resp.StatusCode) {
			return url, resp, err
		}
		loc := resp.Header.Get("Location")
		newBody := io.NopCloser(bytes.NewBuffer(bodyBytes))

		if strings.HasPrefix(loc, "http") {
			return inner(method, loc, header, newBody, autoRedirect, depth+1)
		}

		if strings.HasPrefix(loc, "/") {
			loc = fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, loc)
			return inner(method, loc, header, newBody, autoRedirect, depth+1)
		}

		dirPath := path.Dir(req.URL.Path)
		loc = fmt.Sprintf("%s://%s%s/%s", req.URL.Scheme, req.URL.Host, dirPath, loc)
		return inner(method, loc, header, newBody, autoRedirect, depth+1)
	}

	return inner(r.method, r.url, r.header, r.body, r.redirect, 0)
}
