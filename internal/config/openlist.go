package config

import (
	"errors"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/strs"
)

type Openlist struct {
	// Token 访问 openlist 接口的密钥, 在 openlist 管理后台获取
	Token string `yaml:"token"`
	// Host openlist 访问地址（如果 openlist 使用本地代理模式, 则这个地址必须配置公网可访问地址）
	Host string `yaml:"host"`
}

func (a *Openlist) Init() error {
	if strs.AnyEmpty(a.Token) {
		return errors.New("openlist.token 配置不能为空")
	}
	if strs.AnyEmpty(a.Host) {
		return errors.New("openlist.host 配置不能为空")
	}
	return nil
}
