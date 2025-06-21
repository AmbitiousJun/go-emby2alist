package m3u8_test

import (
	"log"
	"testing"

	"github.com/AmbitiousJun/go-emby2openlist/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/internal/service/m3u8"
)

func TestPlaylistCache(t *testing.T) {
	config.ReadFromFile("../../../config.yml")
	info := m3u8.Info{
		OpenlistPath: "/运动/安小雨跳绳课 (2021)/安小雨跳绳课.S01E01.3000次.25分钟.1080p.mp4",
		TemplateId:   "FHD",
	}

	// 注册 playlist
	m3u8.PushPlaylistAsync(info)

	// 获取 playlist
	m3uContent, ok := m3u8.GetPlaylist(info.OpenlistPath, info.TemplateId, true, true, "", "")
	if !ok {
		log.Fatal("获取 m3u 失败")
	}
	log.Println(m3uContent)

	// 获取 ts
	log.Printf("\n\n\n")
	log.Println("获取 162 ts: ")
	log.Println(m3u8.GetTsLink(info.OpenlistPath, info.TemplateId, 162))

	// 获取 ts
	log.Printf("\n\n\n")
	log.Println("获取 150 ts: ")
	log.Println(m3u8.GetTsLink(info.OpenlistPath, info.TemplateId, 150))
}
