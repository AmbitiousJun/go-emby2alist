package config

import "github.com/AmbitiousJun/go-emby2openlist/internal/setup"

// Log 日志配置
type Log struct {
	DisableColor bool `yaml:"disable-color"` // 是否禁用彩色日志输出
}

// Init 配置初始化
func (lc *Log) Init() error {
	setup.LogColorEnbale = !lc.DisableColor
	return nil
}
