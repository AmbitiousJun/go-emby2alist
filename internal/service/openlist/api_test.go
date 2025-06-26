package openlist_test

import (
	"log"
	"net/http"
	"testing"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/service/openlist"
)

func TestFetch(t *testing.T) {
	err := config.ReadFromFile("../../../config.yml")
	if err != nil {
		t.Error(err)
		return
	}

	var res openlist.FsList
	err = openlist.Fetch("/api/fs/list", http.MethodPost, nil, map[string]any{
		"refresh":  true,
		"password": "",
		"path":     "/",
	}, &res)
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("请求成功, data: %v", res)
}
