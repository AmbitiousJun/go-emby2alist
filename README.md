# go-emby2alist

使用 Go 语言编写的网盘直链反向代理服务，为 Emby + Alist 组合提供更好的使用体验。

> 本项目只专注服务于网盘，且不打算兼容 Jellyfin/Plex（这两个我也用过，说实话真不如 Emby），主打一个易用和轻巧，如果你的 Emby 有其他类型的资源，推荐使用[功能更完善的反向代理服务](https://github.com/bpking1/embyExternalUrl)



**功能**：

- 静态资源 301 重定向
- Alist 网盘原画直链播放
- Alist 网盘转码直链播放
- websocket 代理
- 客户端防转码（转容器）
- 字幕缓存
- 缓存中间件，实际使用体验不会比直连源服务器差



**已测试并支持的客户端**：

| 客户端                           | 已知问题                                                     |
| -------------------------------- | ------------------------------------------------------------ |
| `Emby Web`                       | 无法正常播放 Alist 转码 m3u8 直链                            |
| `Emby for macOS`，`Emby for iOS` | 基本的操作都正常，~~由于本人没这两个客户端的高级版~~，无法测试播放功能 |
| `Emby for Android`               | 使用安卓 TV 测试，功能大部分正常，Alist 转码 m3u8 直链可播放，可保存进度，但是无法恢复播放，且在播放时会出现跳帧的情况（直链过期） |
| `Fileball`                       | 所有功能可用，缺点是只支持 IOS                               |
| `Infuse`                         | 功能大部分正常，只支持播放原画直链（在设置中将缓存方式设置为`不缓存`可有效防止频繁请求） |
| `VidHub`                         | 所有功能可用，缺点同样是只支持苹果全家桶（~~并且收费~~）     |
| `音流 StreamMusic`               | 直连模式正常可用（~~媒体库模式没测试，个人觉得没必要~~）     |



## 使用 DockerCompose 部署安装

1. 获取代码

```shell
git clone https://ghproxy.cc/https://github.com/AmbitiousJun/go-emby2alist
```

2. 拷贝配置

```shell
cp config-example.yml config.yml
```

3. 根据自己的服务器配置好 `config.yml` 文件
4. 编译并运行容器

```shell
docker-compose up -d --build
```

5. 浏览器访问服务器 ip + 端口 `8095`，开始使用

   > 如需要自定义端口，在第四步编译之前，修改 `docker-compose.yml` 文件中的 `8095:8095` 为 `[自定义端口]:8095` 即可

6. 修改配置的时候需要重新启动容器

```shell
docker-compose down
# 修改 config.yml ...
docker-compose up -d
```

7. 版本更新

```shell
docker-compose down
git pull
docker-compose up -d --build
```

8. 清除过时的 Docker 镜像

```shell
docker image prune -f
```



## 已知问题

1. m3u8 直链兼容性不佳
1. 播放有字幕的资源，第一次播放时，无法阻止 Emby 消耗服务器流量