package emby

// MsInfo MediaSourceId 解析信息
type MsInfo struct {
	Empty            bool   // 传递的 id 是否是个空值
	Transcode        bool   // 是否请求转码的资源
	OriginId         string // 原始 MediaSourceId
	RawId            string // 未经过解析的原始请求 Id
	TemplateId       string // alist 中转码资源的模板 id
	Format           string // 转码资源的格式, 比如：1920x1080
	SourceNamePrefix string // 转码资源名称前缀
	AlistPath        string // 资源在 alist 中的地址
}

// ItemInfo emby 资源 item 解析信息
type ItemInfo struct {
	Id              string     // item id
	MsInfo          MsInfo     // MediaSourceId 解析信息
	ApiKey          string     // emby 接口密钥
	ApiKeyType      ApiKeyType // emby 接口密钥类型
	PlaybackInfoUri string     // item 信息查询接口 uri, 通过源服务器查询
}
