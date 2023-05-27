package main

import (
	"cni-demo/cni"
	"cni-demo/tools/helper"
	"cni-demo/tools/skel"
	"cni-demo/tools/utils"
	"errors"
	"fmt"

	_ "cni-demo/plugins/hostgw"
	_ "cni-demo/plugins/ipip"
	_ "cni-demo/plugins/vxlan/vxlan"
	_ "cni-demo/plugins/xvlan/ipvlan"
	_ "cni-demo/plugins/xvlan/macvlan"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

// cmdAdd 函数用于处理 CNI ADD 操作，主要用于设置网络接口
func cmdAdd(args *skel.CmdArgs) error {
	// 记录日志，表示进入 cmdAdd 函数
	utils.WriteLog("进入到 cmdAdd")
	// 输出临时日志，打印 args
	helper.TmpLogArgs(args)

	// 从 args 里把 config 给捞出来
	pluginConfig := helper.GetConfigs(args)
	if pluginConfig == nil {
		errMsg := fmt.Sprintf("add: 从 args 中获取 plugin config 失败, config: %s", string(args.StdinData))
		utils.WriteLog(errMsg)
		return errors.New(errMsg)
	}

	// 获取 CNI 模式和版本信息
	mode, cniVersion := helper.GetBaseInfo(pluginConfig)
	if pluginConfig.CNIVersion == "" {
		pluginConfig.CNIVersion = cniVersion
	}

	// 将 args 和 configs 以及要使用的插件模式都传给 cni manager
	cniManager := cni.
		GetCNIManager().
		SetBootstrapConfigs(pluginConfig).
		SetBootstrapArgs(args).
		SetBootstrapCNIMode(mode)
	if cniManager == nil {
		utils.WriteLog("cni 插件未初始化完成")
		return errors.New("cni plugins register failed")
	}

	// 启动对应 mode 的插件开始设置乱七八糟的网卡等
	err := cniManager.BootstrapCNI()
	if err != nil {
		utils.WriteLog("设置 cni 失败: ", err.Error())
		return err
	}

	// 将结果打印到标准输出
	err = cniManager.PrintResult()
	if err != nil {
		utils.WriteLog("打印 cni 执行结果失败: ", err.Error())
		return err
	}
	return nil
}

// cmdDel 函数用于处理 CNI DEL 操作，主要用于清理网络接口
func cmdDel(args *skel.CmdArgs) error {
	utils.WriteLog("进入到 cmdDel")
	helper.TmpLogArgs(args)

	// 从 args 中获取 plugin config
	pluginConfig := helper.GetConfigs(args)
	if pluginConfig == nil {
		errMsg := fmt.Sprintf("del: 从 args 中获取 plugin config 失败, config: %s", string(args.StdinData))
		utils.WriteLog(errMsg)
		return errors.New(errMsg)
	}
	mode, _ := helper.GetBaseInfo(pluginConfig)

	// 设置卸载参数
	cniManager := cni.
		GetCNIManager().
		SetUnmountConfigs(pluginConfig).
		SetUnmountArgs(args).
		SetUnmountCNIMode(mode)

	// 进行卸载操作
	return cniManager.UnmountCNI()
}

// cmdCheck 函数用于处理 CNI CHECK 操作，主要用于检查网络接口状态
func cmdCheck(args *skel.CmdArgs) error {
	utils.WriteLog("进入到 cmdCheck")
	helper.TmpLogArgs(args)

	// 从 args 中获取 plugin config
	pluginConfig := helper.GetConfigs(args)
	if pluginConfig == nil {
		errMsg := fmt.Sprintf("check: 从 args 中获取 plugin config 失败, config: %s", string(args.StdinData))
		utils.WriteLog(errMsg)
		return errors.New(errMsg)
	}
	mode, _ := helper.GetBaseInfo(pluginConfig)

	// 设置检查参数
	cniManager := cni.
		GetCNIManager().
		SetCheckConfigs(pluginConfig).
		SetCheckArgs(args).
		SetCheckCNIMode(mode)

	// 进行检查操作
	return cniManager.CheckCNI()

}

func main() {
	// 注册 CNI 插件的三个主要操作（Add, Check, Delete）和版本信息
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("cni-demo"))
}
