package alist

import "net/http"

// FetchInfo 请求 alist 资源需要的参数信息
type FetchInfo struct {
	Path                  string      // alist 资源绝对路径
	UseTranscode          bool        // 是否请求转码资源 (只支持视频资源)
	Format                string      // 要请求的转码资源格式, 如: FHD
	TryRawIfTranscodeFail bool        // 如果请求转码资源失败, 是否尝试请求原画资源
	Header                http.Header // 自定义的请求头
}
