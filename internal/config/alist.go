package config

import (
	"errors"
	"strings"
)

type Alist struct {
	// Token 访问 alist 接口的密钥, 在 alist 管理后台获取
	Token string `yaml:"token"`
	// Host alist 访问地址（如果 alist 使用本地代理模式, 则这个地址必须配置公网可访问地址）
	Host string `yaml:"host"`
}

func (a *Alist) Init() error {
	if a.Token = strings.TrimSpace(a.Token); a.Token == "" {
		return errors.New("alist.token 配置不能为空")
	}
	if a.Host = strings.TrimSpace(a.Host); a.Host == "" {
		return errors.New("alist.host 配置不能为空")
	}
	return nil
}
