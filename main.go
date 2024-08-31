package main

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/web"
	"log"
)

func main() {
	log.Println("正在加载配置...")
	if err := config.ReadFromFile("config.yml"); err != nil {
		log.Fatal("加载失败", err)
	}

	log.Println("正在启动服务, 监听端口 8095...")
	if err := web.Listen(8095); err != nil {
		log.Fatal(err)
	}
}
