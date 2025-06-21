package openlist

var (

	// langDisplayNames 将 openlist 的字幕代码转换成对应名称
	langDisplayNames = map[string]string{
		"chi": "简体中文",
		"eng": "English",
		"jpn": "日本語",
	}
)

// SubLangDisplayName 将 lang 转换成对应名称
func SubLangDisplayName(lang string) string {
	if name, ok := langDisplayNames[lang]; ok {
		return name
	}
	return lang
}
