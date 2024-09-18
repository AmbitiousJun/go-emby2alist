package urls_test

import (
	"log"
	"testing"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/urls"
)

func TestAppendUrlArgs(t *testing.T) {
	rawUrl := "http://localhost:8095/emby/Items/2008/PlaybackInfo?reqformat=json"
	res := urls.AppendArgs(rawUrl, "ambitious", "jun", "Static", "true", "unvalid")
	log.Println("拼接后的结果: ", res)
}
