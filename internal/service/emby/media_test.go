package emby_test

import (
	"log"
	"regexp"
	"strings"
	"testing"
)

func TestMatchItemId(t *testing.T) {
	var itemIdRegex = regexp.MustCompile(`(?:/emby)?/[^/]+/(\d+)/`)
	str := "/emby/Items/2008/PlaybackInfo"
	res := itemIdRegex.FindStringSubmatch(str)
	log.Println(res[1])
	str = strings.ReplaceAll(str, "/emby", "")
	res = itemIdRegex.FindStringSubmatch(str)
	log.Println(res[1])
}
