package urls

import (
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
)

// IsRemote 检查一个地址是否是远程地址
func IsRemote(path string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	return u.Host != ""
}

// TransferSlash 将传递的路径的斜杠转换为正斜杠
//
// 如果传递的参数不是一个路径, 不作任何处理
func TransferSlash(p string) string {
	if strs.AnyEmpty(p) {
		return p
	}
	_, err := url.Parse(p)
	if err != nil {
		return p
	}
	return strings.ReplaceAll(p, `\`, `/`)
}

// ResolveResourceName 解析一个资源 url 的名称
//
// 比如 http://example.com/a.txt?a=1&b=2 会返回 a.txt
func ResolveResourceName(resUrl string) string {
	u, err := url.Parse(resUrl)
	if err != nil {
		return resUrl
	}
	return filepath.Base(u.Path)
}

// ReplaceAll 类似于 strings.ReplaceAll
//
// 区别在于可以一次性传入多个子串进行替换
func ReplaceAll(rawUrl string, oldNews ...string) string {
	if len(oldNews) < 2 {
		return rawUrl
	}
	for i := 0; i < len(oldNews)-1; i += 2 {
		rawUrl = strings.ReplaceAll(rawUrl, oldNews[i], oldNews[i+1])
	}
	return rawUrl
}

// AppendArgs 往 url 中添加 query 参数
//
// 添加参数按照键值对的顺序依次传递到函数中,
// 仅出现偶数个参数才会成功匹配出一个 query 参数
//
// 如果在拼接的过程中出现任何异常, 会返回 rawUrl 而不作任何修改
func AppendArgs(rawUrl string, kvs ...string) string {
	if len(kvs) < 2 {
		return rawUrl
	}

	u, err := url.Parse(rawUrl)
	if err != nil {
		log.Printf("AppendUrlArgs 转换 rawUrl 时出现异常: %v", err)
		return rawUrl
	}

	q := u.Query()
	for i := 0; i < len(kvs)-1; i += 2 {
		q.Set(kvs[i], kvs[i+1])
	}
	u.RawQuery = q.Encode()
	return u.String()
}
