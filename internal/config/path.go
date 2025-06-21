package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
)

type Path struct {
	// Emby2Openlist Emby 的路径前缀映射到 Openlist 的路径前缀, 两个路径使用 : 符号隔开
	Emby2Openlist []string `yaml:"emby2openlist"`

	// emby2OpenlistArr 根据 Emby2Openlist 转换成路径键值对数组
	emby2OpenlistArr [][2]string
}

func (p *Path) Init() error {
	p.emby2OpenlistArr = make([][2]string, 0)
	for _, e2a := range p.Emby2Openlist {
		arr := strings.Split(e2a, ":")
		if len(arr) != 2 {
			return fmt.Errorf("path.emby2openlist 配置错误, %s 无法根据 ':' 进行分割", e2a)
		}
		p.emby2OpenlistArr = append(p.emby2OpenlistArr, [2]string{arr[0], arr[1]})
	}
	return nil
}

// MapEmby2Openlist 将 emby 路径映射成 openlist 路径
func (p *Path) MapEmby2Openlist(embyPath string) (string, bool) {
	for _, cfg := range p.emby2OpenlistArr {
		ep, ap := cfg[0], cfg[1]
		if strings.HasPrefix(embyPath, ep) {
			log.Printf(colors.ToGray("命中 emby2openlist 路径映射: %s => %s (如命中错误, 请将正确的映射配置前移)"), ep, ap)
			return strings.Replace(embyPath, ep, ap, 1), true
		}
	}
	return "", false
}
