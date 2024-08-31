package urls

import (
	"log"
	"net/url"
	"strings"
)

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
