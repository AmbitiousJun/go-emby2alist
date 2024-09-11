package main

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/web"
	"log"
)

const CurrentVersion = "go-emby2alist => 1.0.5-beta-v1"
const RepoArr = "https://github.com/AmbitiousJun/go-emby2alist"

func main() {
	printBanner()
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

func printBanner() {
	log.Println(`
                                  _           ____       _ _     _   
  __ _  ___         ___ _ __ ___ | |__  _   _|___ \ __ _| (_)___| |_ 
 / _| |/ _ \ _____ / _ \ '_ | _ \| '_ \| | | | __) / _| | | / __| __|
| (_| | (_) |_____|  __/ | | | | | |_) | |_| |/ __/ (_| | | \__ \ |_ 
 \__, |\___/       \___|_| |_| |_|_.__/ \__, |_____\__,_|_|_|___/\__|
 |___/                                  |___/                        
	`)
}
