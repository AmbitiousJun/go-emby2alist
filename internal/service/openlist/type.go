package openlist

import "net/http"

// FetchInfo 请求 openlist 资源需要的参数信息
type FetchInfo struct {
	Path                  string      // openlist 资源绝对路径
	UseTranscode          bool        // 是否请求转码资源 (只支持视频资源)
	Format                string      // 要请求的转码资源格式, 如: FHD
	TryRawIfTranscodeFail bool        // 如果请求转码资源失败, 是否尝试请求原画资源
	Header                http.Header // 自定义的请求头
}

// Resource openlist 资源信息封装
type Resource struct {
	Url       string         // 资源远程路径
	Subtitles []SubtitleInfo // 字幕信息
}

// SubtitleInfo 资源内嵌的字幕信息
type SubtitleInfo struct {
	Lang string // 字幕语言
	Url  string // 字幕远程路径
}
