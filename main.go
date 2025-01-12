package main

import (
	"log"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/web"
)

// const CurrentVersion = "v1.4.0"
const CurrentVersion = "beta-20250112.01"
const RepoAddr = "https://github.com/AmbitiousJun/go-emby2alist"

func main() {
	printBanner()

	log.Println("正在加载配置...")
	if err := config.ReadFromFile("config.yml"); err != nil {
		log.Fatal(err)
	}

	log.Println(colors.ToBlue("正在启动服务..."))
	if err := web.Listen(); err != nil {
		log.Fatal(colors.ToRed(err.Error()))
	}
}

func printBanner() {
	log.Printf(colors.ToYellow(`
                                  _           ____       _ _     _   
  __ _  ___         ___ _ __ ___ | |__  _   _|___ \ __ _| (_)___| |_ 
 / _| |/ _ \ _____ / _ \ '_ | _ \| '_ \| | | | __) / _| | | / __| __|
| (_| | (_) |_____|  __/ | | | | | |_) | |_| |/ __/ (_| | | \__ \ |_ 
 \__, |\___/       \___|_| |_| |_|_.__/ \__, |_____\__,_|_|_|___/\__|
 |___/                                  |___/                        
 
 Repository: %s
    Version: %s
	`), RepoAddr, CurrentVersion)
}
