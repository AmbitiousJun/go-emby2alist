package jsons_test

import (
	"encoding/json"
	"log"
	"strconv"
	"testing"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/jsons"
)

func TestMarshal(t *testing.T) {
	log.Println(jsons.FromValue("Ambitious"))
	log.Println(jsons.FromValue(true))
	log.Println(jsons.FromValue(23))
	log.Println(jsons.FromValue(3.14159))
	log.Println(jsons.FromValue(nil))

	arr := []any{"Ambitious", true, 23, 3.14159, nil}
	log.Println(jsons.FromArray(arr))

	m := map[string]any{"1": arr}
	log.Println(jsons.FromObject(m))

	arr = append(arr, map[string]any{"Path": "/a/b/c", "Age": 18, "Name": nil})
	log.Println(jsons.FromArray(arr))
}

func TestUnmarshal(t *testing.T) {
	str := `{"Test": null,"MediaSources":[{"Size":1297828216,"SupportsDirectPlay":false,"TranscodingContainer":"ts","Container":"mp4","Name":"(SD_720x480) 1080p HEVC","Protocol":"File","Formats":[],"TranscodingSubProtocol":"hls","SupportsProbing":false,"HasMixedProtocols":false,"RequiresLooping":false,"Id":"bd083e9d70f3b7322f43fcab2a3dea13","Path":"/data/show/Y/院人全年无休计划 (2023)/Season 2/No.Days.Off.All.Year.S02E01.2024.WEB-DL.1080p.H265.AAC-01案：沉默的三面羊Ⅰ（上）.mp4","SupportsDirectStream":false,"RequiresClosing":false,"Bitrate":2270287,"RequiredHttpHeaders":{},"AddApiKeyToDirectStreamUrl":false,"DefaultAudioStreamIndex":1,"IsRemote":false,"SupportsTranscoding":true,"ReadAtNativeFramerate":false,"TranscodingUrl":"/videos/4005/stream?IsPlayback=false&MaxStreamingBitrate=7000000&X-Emby-Client-Version=4.7.13.0&reqformat=json&MediaSourceId=bd083e9d70f3b7322f43fcab2a3dea13&AutoOpenLiveStream=false&X-Emby-Token=20a4b90f62bc43d9a189a36c71784d7b&StartTimeTicks=0&X-Emby-Device-Id=TW96aWxsYS81LjAgKE1hY2ludG9zaDsgSW50ZWwgTWFjIE9TIFggMTBfMTVfNykgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzExNC4wLjAuMCBTYWZhcmkvNTM3LjM2fDE2ODkxNDkxNjU5NzA1&X-Emby-Client=Emby%20Web&UserId=607606506bab49829edc8e45873f374f&X-Emby-Device-Name=Chrome%20macOS&X-Emby-Language=zh-cn&Static=true&video_preview_format=SD","Type":"Default","IsInfiniteStream":false,"ItemId":"4005","MediaStreams":[{"RealFrameRate":25,"ColorPrimaries":"bt709","IsExternal":false,"ColorTransfer":"bt709","Width":1920,"Protocol":"File","Level":120,"IsAnamorphic":false,"IsInterlaced":false,"IsDefault":true,"AspectRatio":"16:9","TimeBase":"1/90000","IsForced":false,"Language":"und","BitDepth":8,"BitRate":2073001,"SupportsExternalStream":false,"CodecTag":"hev1","AverageFrameRate":25,"ExtendedVideoSubTypeDescription":"None","AttachmentSize":0,"RefFrames":1,"Height":1080,"IsTextSubtitleStream":false,"Codec":"hevc","VideoRange":"SDR","Index":0,"ExtendedVideoType":"None","ColorSpace":"bt709","Type":"Video","IsHearingImpaired":false,"Profile":"Main","PixelFormat":"yuv420p","DisplayTitle":"1080p HEVC","ExtendedVideoSubType":"None"},{"SampleRate":44100,"IsExternal":false,"Protocol":"File","IsInterlaced":false,"IsDefault":true,"TimeBase":"1/44100","Channels":2,"IsForced":false,"Language":"und","BitRate":189588,"SupportsExternalStream":false,"CodecTag":"mp4a","ExtendedVideoSubTypeDescription":"None","AttachmentSize":0,"IsTextSubtitleStream":false,"Codec":"aac","Index":1,"ExtendedVideoType":"None","ChannelLayout":"stereo","Type":"Audio","IsHearingImpaired":false,"Profile":"LC","DisplayTitle":"AAC stereo (默认)","ExtendedVideoSubType":"None"}],"RunTimeTicks":45732640000,"RequiresOpening":false}],"PlaySessionId":"51dcf8dadc4d4b81b52613b88cd4e93f"}`
	item, err := jsons.New(str)
	if err != nil {
		t.Fatal(err)
		return
	}
	log.Println("反序列化成功")

	log.Println("当前 json 类型: ", item.Type())
	item.Attr("Test").Set("This val has modified by test program")

	item.Attr("MediaSources").Idx(0).Attr("DefaultAudioStreamIndex").Set("😁")

	log.Println("重新序列化: ", item.String())
}

func TestMap(t *testing.T) {
	item := jsons.FromArray([]any{1, 2, 1, 3, 8})
	res := item.Map(func(val *jsons.Item) any { return "😄" + strconv.Itoa(val.Ti().Val().(int)) })
	log.Println("转换完成后的数组: ", res)
}

func TestNativeUnmarshal(t *testing.T) {
	str := `aaa`
	var dest string
	if err := json.Unmarshal([]byte(str), &dest); err != nil {
		t.Fatal(err)
		return
	}
	log.Println(dest)

}

func TestNativeMarshal(t *testing.T) {
	str := `aaa`
	if res, err := json.Marshal(str); err != nil {
		t.Fatal(err)
		return
	} else {
		log.Println(string(res))
	}
}
