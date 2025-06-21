package m3u8

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/urls"
)

const (

	// MaxPlaylistNum 在内存中最多维护的 m3u8 列表个数
	// 超出则淘汰最久没有读取的一个
	MaxPlaylistNum = 10

	// PreChanSize 预处理通道大小, 塞满时从头部开始淘汰
	PreChanSize = 1000
)

func init() {
	go loopMaintainPlaylist()
}

// GetPlaylist 获取 m3u 播放列表, 返回 m3u 文本
var GetPlaylist func(openlistPath, templateId string, proxy, main bool, routePrefix, clientApiKey string) (string, bool)

// GetTsLink 获取 m3u 播放列表中的某个 ts 链接
var GetTsLink func(openlistPath, templateId string, idx int) (string, bool)

// GetSubtitleLink 获取字幕链接
var GetSubtitleLink func(openlistPath, templateId, subName string) (string, bool)

// preMaintainInfoChan 预处理通道
//
// 外界将需要维护的信息放到这个通道中, 由 goroutine 单线程维护内存
var preMaintainInfoChan = make(chan Info, PreChanSize)

// preChanHandlingGroup 维护预处理通道的处理状态
//
// 当有新的任务加入通道时, group + 1
// 当 gouroutine 处理完一个任务时, group - 1
var preChanHandlingGroup = sync.WaitGroup{}

// PushPlaylistAsync 将一个 openlist 转码资源异步缓存到内存中
func PushPlaylistAsync(info Info) {
	if info.OpenlistPath == "" || info.TemplateId == "" {
		return
	}
	info = Info{OpenlistPath: info.OpenlistPath, TemplateId: info.TemplateId}
	preChanHandlingGroup.Add(1)
	doneOnce := sync.OnceFunc(preChanHandlingGroup.Done)
	go func() {
		for {
			select {
			case preMaintainInfoChan <- info:
				return
			default:
				<-preMaintainInfoChan
				// 从通道中淘汰旧元素, 通道总大小不会改变
				doneOnce()
			}
		}
	}()
}

// loopMaintainPlaylist 由单独的 goroutine 执行
//
// 维护内存中的 m3u8 播放列表
func loopMaintainPlaylist() {
	// map 记录播放列表, 用于快速响应客户端
	infoMap := map[string]*Info{}
	// arr 记录播放列表, 便于实现淘汰机制
	infoArr := make([]*Info, 0)

	// maintainDuration goroutine 维护 playlist 的间隔
	maintainDuration := time.Minute * 10
	// stopUpdateTimeMillis 超过这个时间未读, playlist 停止更新
	stopUpdateTimeMillis := (maintainDuration + time.Minute).Milliseconds()
	// removeTimeMillis 超过这个时间未读, playlist 被移除
	removeTimeMillis := time.Hour.Milliseconds()

	// publicApiUpdateMutex 对外部暴露的 api 的内部实现中
	// 如果涉及到更新的操作, 需要获取这个锁, 避免频繁请求 openlist
	publicApiUpdateMutex := sync.Mutex{}

	// printErr 打印错误日志
	printErr := func(info *Info, err error) {
		log.Printf(colors.ToRed("playlist 更新失败, path: %s, template: %s, err: %v"), info.OpenlistPath, info.TemplateId, err)
	}

	// calcMapKey 计算 info 在 map 中的 key
	calcMapKey := func(info Info) string {
		return info.OpenlistPath + info.TemplateId
	}

	// beforeNow 判断一个时间是不是在当前时间之前
	beforeNow := func(millis int64) bool {
		return millis < time.Now().UnixMilli()
	}

	// queryInfo 查询内存中的 info 信息
	//
	// 如果内存中 map 已经能查询到 info 信息, 直接返回
	// 否则会等待预处理通道处理完毕后再次判断
	queryInfo := func(openlistPath, templateId string) (info *Info) {
		key := calcMapKey(Info{OpenlistPath: openlistPath, TemplateId: templateId})
		var ok bool
		info, ok = infoMap[key]

		defer func() {
			if info == nil {
				return
			}
			// 如果当前 info 已经停止更新, 则手动触发更新
			if beforeNow(info.LastRead + stopUpdateTimeMillis) {
				publicApiUpdateMutex.Lock()
				defer publicApiUpdateMutex.Unlock()
				if beforeNow(info.LastRead + stopUpdateTimeMillis) {
					if err := info.UpdateContent(); err != nil {
						printErr(info, err)
						info = nil
					}
				}
			}
			// 更新最后读取时间
			info.LastRead = time.Now().UnixMilli()
		}()

		if ok {
			return
		}

		// 等待预处理通道处理完毕
		preChanHandlingGroup.Wait()

		info, ok = infoMap[key]
		if ok {
			return
		}
		return nil
	}

	GetPlaylist = func(openlistPath, templateId string, proxy, main bool, routePrefix, clientApiKey string) (string, bool) {
		info := queryInfo(openlistPath, templateId)
		if info == nil {
			return "", false
		}
		if proxy {
			return info.ProxyContent(main, routePrefix, clientApiKey), true
		}
		return info.Content(), true
	}

	GetTsLink = func(openlistPath, templateId string, idx int) (string, bool) {
		info := queryInfo(openlistPath, templateId)
		if info == nil {
			return "", false
		}
		return info.GetTsLink(idx)
	}

	GetSubtitleLink = func(openlistPath, templateId, subName string) (string, bool) {
		info := queryInfo(openlistPath, templateId)
		if info == nil {
			return "", false
		}
		for _, subInfo := range info.Subtitles {
			curSubName := urls.ResolveResourceName(subInfo.Url)
			if curSubName == subName {
				return subInfo.Url, true
			}
		}
		return "", false
	}

	// removeInfo 删除内存中的 info 信息
	removeInfo := func(key string) {
		info, ok := infoMap[key]
		if !ok {
			return
		}
		delete(infoMap, key)
		for i, arrInfo := range infoArr {
			if arrInfo == info {
				infoArr = append(infoArr[:i], infoArr[i+1:]...)
				break
			}
		}
	}

	// updateAll 更新内存中的 info 信息
	//
	// 如果 lastRead 不满足条件, 被淘汰
	updateAll := func() {
		// 复制一份 arr
		cpArr := append(([]*Info)(nil), infoArr...)
		tot, active := len(cpArr), 0

		for _, info := range cpArr {
			key := calcMapKey(Info{OpenlistPath: info.OpenlistPath, TemplateId: info.TemplateId})

			// 长时间未读, 移除
			if beforeNow(info.LastUpdate + removeTimeMillis) {
				removeInfo(key)
				log.Printf(colors.ToGray("playlist 长时间未被更新, 已移除, openlistPath: %s, templateId: %s"), info.OpenlistPath, info.TemplateId)
				tot--
				continue
			}

			// 超过指定时间未读, 不更新
			if beforeNow(info.LastRead + stopUpdateTimeMillis) {
				continue
			}

			// 如果更新失败, 移除
			active++
			if err := info.UpdateContent(); err != nil {
				printErr(info, err)
				removeInfo(key)
				tot--
				active--
			}
		}

		if len(cpArr) > 0 {
			log.Printf(colors.ToPurple("当前正在维护的 playlist 个数: %d, 活跃个数: %d"), tot, active)
		}
	}

	// addInfo 添加 info 到内存中
	addInfo := func(preInfo Info) {
		if preInfo.OpenlistPath == "" || preInfo.TemplateId == "" {
			return
		}
		key := calcMapKey(preInfo)

		// 如果内存已存在 key, 复用
		info, exist := infoMap[key]
		if !exist {
			info = &preInfo
		}

		// 初始化 Info 信息, 并更新
		if err := info.UpdateContent(); err != nil {
			printErr(info, err)
			removeInfo(key)
			return
		}
		info.LastRead = time.Now().UnixMilli()

		// 维护到内存中
		if !exist {
			infoMap[key] = info
			infoArr = append(infoArr, info)
		}

		if len(infoArr) <= MaxPlaylistNum {
			return
		}
		// 内存满, 淘汰旧内存
		sort.Slice(infoArr, func(i, j int) bool {
			return infoArr[i].LastRead < infoArr[j].LastRead
		})
		toDeletes := make([]*Info, len(infoArr)-MaxPlaylistNum)
		copy(toDeletes, infoArr)
		for _, toDel := range toDeletes {
			removeInfo(calcMapKey(Info{OpenlistPath: toDel.OpenlistPath, TemplateId: toDel.TemplateId}))
			log.Printf(colors.ToGray("playlist 被淘汰并从内存中移除, openlistPath: %s, templateId: %s"), toDel.OpenlistPath, toDel.TemplateId)
		}
	}

	// 定时维护一次内存中的数据
	t := time.NewTicker(maintainDuration)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			updateAll()
		case preInfo := <-preMaintainInfoChan:
			addInfo(preInfo)
			preChanHandlingGroup.Done()
		}
	}

}
