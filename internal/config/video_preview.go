package config

type VideoPreview struct {
	// Enable 是否开启网盘转码链接代理
	Enable bool `yaml:"enable"`
	// Containers 对哪些容器使用网盘转码链接代理
	Containers []string `yaml:"containers"`

	// containerMap 依据 Containers 初始化该 map, 便于后续快速判断
	containerMap map[string]struct{}
}

func (vp *VideoPreview) Init() error {
	vp.containerMap = make(map[string]struct{})
	for _, container := range vp.Containers {
		vp.containerMap[container] = struct{}{}
	}
	return nil
}

// ContainerValid 判断某个视频容器是否启用代理
//
//	@param container
//	@return bool
func (vp *VideoPreview) ContainerValid(container string) bool {
	_, ok := vp.containerMap[container]
	return ok
}
