package config

type Openlist struct {
	// Token 访问 openlist 接口的密钥, 在 openlist 管理后台获取
	Token string `yaml:"token"`
	// Host openlist 访问地址（如果 openlist 使用本地代理模式, 则这个地址必须配置公网可访问地址）
	Host string `yaml:"host"`
}

func (a *Openlist) Init() error {
	return nil
}
