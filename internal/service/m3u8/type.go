package m3u8

import "github.com/AmbitiousJun/go-emby2alist/internal/service/openlist"

// ParentHeadComments 记录文件头注释
var ParentHeadComments = map[string]struct{}{
	"#EXTM3U": {}, "#EXT-X-VERSION": {}, "#EXT-X-MEDIA-SEQUENCE": {},
	"#EXT-X-TARGETDURATION": {}, "#EXT-X-MEDIA": {}, "#EXT-X-INDEPENDENT-SEGMENTS": {},
	"#EXT-X-STREAM-INF": {},
}

// ParentTailComments 记录文件尾注释
var ParentTailComments = map[string]struct{}{
	"#EXT-X-ENDLIST": {},
}

// 响应头中，有效的 m3u8 Content-Type 属性
var ValidM3U8Contents = map[string]struct{}{
	"application/vnd.apple.mpegurl": {},
	"application/x-mpegurl":         {},
	"audio/x-mpegurl":               {},
	"application/octet-stream":      {},
}

// Info 记录一个 m3u8 相关信息
type Info struct {
	AlistPath     string                  // 资源在 alist 中的绝对路径
	TemplateId    string                  // 转码资源模板 id
	Subtitles     []openlist.SubtitleInfo // 字幕信息, 如果一个资源是含有字幕的, 会返回变体 m3u8
	RemoteBase    string                  // 远程 m3u8 地址前缀
	HeadComments  []string                // 头注释信息
	TailComments  []string                // 尾注释信息
	RemoteTsInfos []*TsInfo               // 远程的 ts URL 列表, 用于重定向

	// LastRead 客户端最后读取的时间戳 (毫秒)
	//
	// 超过 30 分钟未读取, 程序停止更新;
	// 超过 12 小时未读取, m3u info 被移除
	LastRead int64

	// LastUpdate 程序最后的更新时间戳 (毫秒)
	//
	// 客户端来读取时, 如果 m3u info 已经超过 10 分钟没有更新了
	// 触发更新机制之后, 再返回最新的地址
	LastUpdate int64
}

// TsInfo 记录一个 ts 相关信息
type TsInfo struct {
	Comments []string // 注释信息
	Url      string   // 远程流请求地址
}

// ProxyParams 代理请求接收参数
type ProxyParams struct {
	AlistPath  string `form:"alist_path"`
	TemplateId string `form:"template_id"`
	Remote     string `form:"remote"`
	Type       string `form:"type"`
	ApiKey     string `form:"api_key"`
	IdxStr     string `form:"idx"`
}
