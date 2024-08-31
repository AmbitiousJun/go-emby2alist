package alist_test

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/service/alist"
	"log"
	"net/http"
	"testing"
)

func TestFetch(t *testing.T) {
	err := config.ReadFromFile("../../../config.yml")
	if err != nil {
		t.Error(err)
		return
	}
	res := alist.Fetch("/api/fs/list", http.MethodPost, map[string]interface{}{
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
