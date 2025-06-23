package openlist

import (
	"encoding/json"
	"net/http"
)

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
	Url       string                    // 资源远程路径
	Subtitles []TranscodingSubtitleInfo // 字幕信息
}

// TranscodingSubtitleInfo 转码资源内嵌的字幕信息
type TranscodingSubtitleInfo struct {
	Lang   string `json:"language"` // 字幕语言
	Url    string `json:"url"`      // 字幕远程路径
	Status string `json:"status"`   // 字幕状态
}

// TranscodingVideoInfo 转码资源内嵌的视频信息
type TranscodingVideoInfo struct {
	Stage          string `json:"stage"`           // 转码阶段
	Status         string `json:"status"`          // 转码状态
	TemplateHeight int    `json:"template_height"` // 转码模板高度
	TemplateId     string `json:"template_id"`     // 转码模板 ID
	TemplateName   string `json:"template_name"`   // 转码模板名称
	TemplateWidth  int    `json:"template_width"`  // 转码模板宽度
	Url            string `json:"url"`             // 转码资源链接
}

// RemoteCommonResult openlist 远程响应的通用结果结构
type RemoteCommonResult struct {
	Code    int             `json:"code"`    // 响应状态码
	Message string          `json:"message"` // 响应消息
	Data    json.RawMessage `json:"data"`    // 响应数据
}

// FsOther /api/fs/other 接口响应数据结构
type FsOther struct {
	Category                    string `json:"category"`                       // 资源分类
	DriveId                     string `json:"drive_id"`                       // 资源所在的云盘 ID
	FileId                      string `json:"file_id"`                        // 资源文件 ID
	MetaNameInvestigationStatus int    `json:"meta_name_investigation_status"` // 元数据名称调查状态
	MetaNamePunishFlag          int    `json:"meta_name_punish_flag"`          // 元数据名称惩罚标志
	PunishFlag                  int    `json:"punish_flag"`                    // 资源惩罚标志
	VideoPreviewPlayInfo        struct {
		Category                        string                    `json:"category"`                            // 资源分类
		LiveTranscodingSubtitleTaskList []TranscodingSubtitleInfo `json:"live_transcoding_subtitle_task_list"` // 转码字幕任务列表
		LiveTranscodingTaskList         []TranscodingVideoInfo    `json:"live_transcoding_task_list"`          // 转码任务列表
		Meta                            struct {
			Duration float64 `json:"duration"` // 资源时长
			Height   int     `json:"height"`   // 资源高度
			Width    int     `json:"width"`    // 资源宽度
		} `json:"meta"` // 元数据
	} `json:"video_preview_play_info"` // 视频预览播放信息
}

// FsGet /api/fs/get 接口响应数据结构
type FsGet struct {
	Name     string `json:"name"`     // 文件名
	Size     int    `json:"size"`     // 文件大小
	IsDir    bool   `json:"is_dir"`   // 是否为文件夹
	Modified string `json:"modified"` // 修改时间
	Sign     string `json:"sign"`     // 文件签名
	Thumb    string `json:"thumb"`    // 缩略图链接
	Type     int    `json:"type"`     // 类型
	RawUrl   string `json:"raw_url"`  // 原始资源链接
	Readme   string `json:"readme"`   // 说明
	Provider string `json:"provider"` // 提供者
	Created  string `json:"created"`  // 创建时间
	HashInfo string `json:"hashinfo"` // 哈希信息
	Header   string `json:"header"`   // 头信息
}

// FsList /api/fs/list 接口响应数据结构
type FsList struct {
	Total    int     `json:"total"`    // 总数
	Readme   string  `json:"readme"`   // 说明
	Write    bool    `json:"write"`    // 是否可写入
	Provider string  `json:"provider"` // 提供者
	Header   string  `json:"header"`   // 头信息
	Content  []FsGet `json:"content"`  // 文件列表
}
