package main

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/web"
	"log"
)

const CurrentVersion = "go-emby2alist => 1.0.0-beta-v2"
const RepoArr = "https://github.com/AmbitiousJun/go-emby2alist"

func main() {
	log.Printf(color.ToPurple("版本号: %s"), CurrentVersion)
	log.Printf(color.ToBlue("仓库地址: %s"), RepoArr)

	log.Println("正在加载配置...")
	if err := config.ReadFromFile("config.yml"); err != nil {
		log.Fatal("加载失败", err)
	}

	log.Println("正在启动服务, 监听端口 8095...")
	if err := web.Listen(8095); err != nil {
		log.Fatal(err)
	}
}
