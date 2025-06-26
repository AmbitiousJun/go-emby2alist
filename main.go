package main

import (
	"log"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/constant"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/web"
)

func main() {
	log.Println("正在加载配置...")
	if err := config.ReadFromFile("config.yml"); err != nil {
		log.Fatal(err)
	}

	printBanner()

	log.Println(colors.ToBlue("正在启动服务..."))
	if err := web.Listen(); err != nil {
		log.Fatal(colors.ToRed(err.Error()))
	}
}

func printBanner() {
	log.Printf(colors.ToYellow(`
                                 _           ___                        _ _     _   
                                | |         |__ \                      | (_)   | |  
  __ _  ___ ______ ___ _ __ ___ | |__  _   _   ) |___  _ __   ___ _ __ | |_ ___| |_ 
 / _| |/ _ \______/ _ \ '_ | _ \| '_ \| | | | / // _ \| '_ \ / _ \ '_ \| | / __| __|
| (_| | (_) |    |  __/ | | | | | |_) | |_| |/ /| (_) | |_) |  __/ | | | | \__ \ |_ 
 \__, |\___/      \___|_| |_| |_|_.__/ \__, |____\___/| .__/ \___|_| |_|_|_|___/\__|
  __/ |                                 __/ |         | |                           
 |___/                                 |___/          |_|                           

 Repository: %s
    Version: %s
	`), constant.RepoAddr, constant.CurrentVersion)
}
