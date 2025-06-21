package path

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/AmbitiousJun/go-emby2openlist/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/internal/service/openlist"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/urls"
)

// OpenlistPathRes 路径转换结果
type OpenlistPathRes struct {

	// Success 转换是否成功
	Success bool

	// Path 转换后的路径
	Path string

	// Range 遍历所有 Openlist 根路径生成的子路径
	Range func() ([]string, error)
}

// Emby2Openlist Emby 资源路径转 Openlist 资源路径
func Emby2Openlist(embyPath string) OpenlistPathRes {
	pathRoutes := strings.Builder{}
	pathRoutes.WriteString("[")
	pathRoutes.WriteString("\n【原始路径】 => " + embyPath)

	embyPath = urls.TransferSlash(embyPath)
	pathRoutes.WriteString("\n\n【Windows 反斜杠转换】 => " + embyPath)

	embyMount := config.C.Emby.MountPath
	openlistFilePath := strings.TrimPrefix(embyPath, embyMount)
	pathRoutes.WriteString("\n\n【移除 mount-path】 => " + openlistFilePath)

	openlistFilePath = urls.Unescape(openlistFilePath)
	pathRoutes.WriteString("\n\n【URL 解码】 => " + openlistFilePath)

	if mapPath, ok := config.C.Path.MapEmby2Openlist(openlistFilePath); ok {
		openlistFilePath = mapPath
		pathRoutes.WriteString("\n\n【命中 emby2openlist 映射】 => " + openlistFilePath)
	}
	pathRoutes.WriteString("\n]")
	log.Printf(colors.ToGray("embyPath 转换路径: %s"), pathRoutes.String())

	rangeFunc := func() ([]string, error) {
		filePath, err := SplitFromSecondSlash(openlistFilePath)
		if err != nil {
			return nil, fmt.Errorf("openlistFilePath 解析异常: %s, error: %v", openlistFilePath, err)
		}

		res := openlist.FetchFsList("/", nil)
		if res.Code != http.StatusOK {
			return nil, fmt.Errorf("请求 openlist fs list 接口异常: %s", res.Msg)
		}

		paths := make([]string, 0)
		content, ok := res.Data.Attr("content").Done()
		if !ok || content.Type() != jsons.JsonTypeArr {
			return nil, fmt.Errorf("openlist fs list 接口响应异常, 原始响应: %v", jsons.NewByObj(res))
		}

		content.RangeArr(func(_ int, value *jsons.Item) error {
			if value.Attr("is_dir").Val() == false {
				return nil
			}
			newPath := fmt.Sprintf("/%s%s", value.Attr("name").Val(), filePath)
			paths = append(paths, newPath)
			return nil
		})

		return paths, nil
	}

	return OpenlistPathRes{
		Success: true,
		Path:    openlistFilePath,
		Range:   rangeFunc,
	}
}

// SplitFromSecondSlash 找到给定字符串 str 中第二个 '/' 字符的位置
// 并以该位置为首字符切割剩余的子串返回
func SplitFromSecondSlash(str string) (string, error) {
	str = urls.TransferSlash(str)
	firstIdx := strings.Index(str, "/")
	if firstIdx == -1 {
		return "", fmt.Errorf("字符串不包含 /: %s", str)
	}

	secondIdx := strings.Index(str[firstIdx+1:], "/")
	if secondIdx == -1 {
		return "", fmt.Errorf("字符串只有单个 /: %s", str)
	}

	return str[secondIdx+firstIdx+1:], nil
}
