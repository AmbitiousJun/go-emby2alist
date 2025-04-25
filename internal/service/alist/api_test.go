package alist_test

import (
	"log"
	"net/http"
	"testing"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/alist"
)

func TestFetch(t *testing.T) {
	err := config.ReadFromFile("../../../config.yml")
	if err != nil {
		t.Error(err)
		return
	}
	res := alist.Fetch("/api/fs/list", http.MethodPost, nil, map[string]any{
		"refresh":  true,
		"password": "",
		"path":     "/",
	})
	if res.Code == http.StatusOK {
		log.Println("请求成功, data: ", res.Data)
	} else {
		log.Println("请求失败, msg: ", res.Msg)
	}
}
