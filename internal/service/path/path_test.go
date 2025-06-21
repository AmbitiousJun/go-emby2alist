package path_test

import (
	"log"
	"testing"

	"github.com/AmbitiousJun/go-emby2openlist/internal/service/path"
)

func TestSplit(t *testing.T) {
	str := `H:\Phim4K\The.Lockdown.2024.2160p.WEB-DL.DDP5.1.DV.HDR.H.265-FLUX.mkv`
	log.Println(path.SplitFromSecondSlash(str))
}
