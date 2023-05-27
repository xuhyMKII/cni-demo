package helper

import (
	"cni-demo/cni"
	"cni-demo/consts"
	"cni-demo/tools/skel"
	"cni-demo/tools/utils"
	"encoding/json"
)

// 输入参数: args *skel.CmdArgs 是从 CNI 插件收到的命令行参数
// 函数功能: 将传入的 args.StdinData JSON 数据解析为 cni.PluginConf 结构体
// 返回值: 解析后的 *cni.PluginConf 结构体指针
func GetConfigs(args *skel.CmdArgs) *cni.PluginConf {
	pluginConfig := &cni.PluginConf{}
	if err := json.Unmarshal(args.StdinData, pluginConfig); err != nil {
		utils.WriteLog("args.StdinData 转 pluginConfig 失败")
		return nil
	}
	// utils.WriteLog("这里的结果是: pluginConfig.Bridge", pluginConfig.Bridge)
	// utils.WriteLog("这里的结果是: pluginConfig.CNIVersion", pluginConfig.CNIVersion)
	// utils.WriteLog("这里的结果是: pluginConfig.Name", pluginConfig.Name)
	// utils.WriteLog("这里的结果是: pluginConfig.Subnet", pluginConfig.Subnet)
	// utils.WriteLog("这里的结果是: pluginConfig.Type", pluginConfig.Type)
	// utils.WriteLog("这里的结果是: pluginConfig.Mode", pluginConfig.Mode)
	return pluginConfig
}

// 输入参数: plugin *cni.PluginConf 是 CNI 插件的配置结构体
// 函数功能: 提取 CNI 插件的工作模式（mode）和 CNI 版本（cniVersion），如果未设置，则分别使用默认值 consts.MODE_HOST_GW 和 "0.3.0"
// 返回值: 工作模式（mode）和 CNI 版本（cniVersion）
func GetBaseInfo(plugin *cni.PluginConf) (mode string, cniVersion string) {
	mode = plugin.Mode
	if mode == "" {
		mode = consts.MODE_HOST_GW
	}
	cniVersion = plugin.CNIVersion
	if cniVersion == "" {
		cniVersion = "0.3.0"
	}
	return mode, cniVersion
}

// 输入参数: args *skel.CmdArgs 是从 CNI 插件收到的命令行参数
// 函数功能: 将传入的 args 中的各项参数（ContainerID、Netns、IfName、Args、Path 和 StdinData）记录到日志中，用于调试和跟踪
func TmpLogArgs(args *skel.CmdArgs) {
	utils.WriteLog(
		"这里的 CmdArgs 是: ", "ContainerID: ", args.ContainerID,
		"Netns: ", args.Netns,
		"IfName: ", args.IfName,
		"Args: ", args.Args,
		"Path: ", args.Path,
		"StdinData: ", string(args.StdinData),
	)
}
