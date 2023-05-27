package cni

import (
	"cni-demo/tools/skel"
	"cni-demo/tools/utils"
	"errors"
	"fmt"

	cniTypes "github.com/containernetworking/cni/pkg/types"
	types "github.com/containernetworking/cni/pkg/types/100"
)

// IPAM 结构体定义了 IPAM 配置，其中包括类型、子网、范围开始、范围结束、网关、地址以及路由等信息。
type IPAM struct {
	Type       string                     `json:"type"`
	Subnet     string                     `json:"subnet"`
	RangeStart string                     `json:"rangeStart"`
	RangeEnd   string                     `json:"rangeEnd"`
	Gateway    string                     `json:"gateway"`
	Addresses  []struct{ Address string } `json:"addresses"`
	Routes     interface{}                `json:"routes"`
}

// PluginConf 结构体定义了插件配置，包括 NetConf（基本信息）、RuntimeConfig（运行时配置）、IPAM（IPAM 配置）、桥接、子网和模式等信息。
type PluginConf struct {
	// NetConf 里头指定了一个 plugin 的最基本的信息, 比如 CNIVersion, Name, Type 等, 当然还有在 containerd 中塞进来的 PrevResult
	cniTypes.NetConf

	// 这个 runtimeConfig 是可以在 /etc/cni/net.d/xxx.conf 中配置一个
	// 类似 "capabilities": {"xxx": true, "yyy": false} 这样的属性
	// 表示说要在运行时开启 xxx 的能力, 不开启 yyy 的能力
	// 然后等容器跑起来之后(或者被拉起来之前)可以直接通过设置环境变量 export CAP_ARGS='{ "xxx": "aaaa", "yyy": "bbbb" }'
	// 来开启或关闭某些能力
	// 然后通过 stdin 标准输入读进来的数据中就会多出一个 RuntimeConfig 属性, 里面就是 runtimeConfig: { "xxx": "aaaa" }
	// 因为 yyy 在 /etc/cni/net.d/xxx.conf 中被设置为了 false
	// 官方使用范例: https://kubernetes.feisky.xyz/extension/network/cni
	// cni 源码中实现: /cni/libcni/api.go:injectRuntimeConfig
	RuntimeConfig *struct {
		TestConfig map[string]interface{} `json:"testConfig"`
	} `json:"runtimeConfig"`

	IPAM *IPAM `json:"ipam"`
	// 这里可以自由定义自己的 plugin 中配置了的参数然后自由处理
	Bridge string `json:"bridge"`
	Subnet string `json:"subnet"`
	Mode   string `json:"mode" default:"host-gw"`
}

var manager *CNIManager

// CNI 接口定义了 CNI 插件的通用方法，包括 Bootstrap（启动）、Unmount（卸载）、Check（检查）和 GetMode（获取模式）。
type CNI interface {
	Bootstrap(
		args *skel.CmdArgs,
		pluginConfig *PluginConf,
	) (*types.Result, error)
	Unmount(
		args *skel.CmdArgs, // 对于卸载或检查来讲, args 可能不同于启动时
		pluginConfig *PluginConf,
	) error
	Check(
		args *skel.CmdArgs, // 对于卸载或检查来讲, args 可能不同于启动时
		pluginConfig *PluginConf,
	) error
	GetMode() string
}

// CNIManager 结构体定义了 CNI 管理器，包括 cniMap（CNI 插件映射）、各种模式（启动、卸载、检查）、各种参数（启动、卸载、检查）、插件配置和结果等属性。
type CNIManager struct {
	cniMap map[string]CNI
	/**
	 * 对于 mode 和 config 来讲
	 * 暂时没听说哪个插件在挂载和卸载以及检查时是不一样的
	 * 不过既然 kubelet 在不同时机时传过来的 args 有可能不一样
	 * 那就先不排除不同时机传进来的 config 不同的这种骚气的操作, 以防万一
	 */
	bootstrapMode string
	unmountMode   string
	checkMode     string

	bootstrapArgs *skel.CmdArgs
	unmountArgs   *skel.CmdArgs
	checkArgs     *skel.CmdArgs

	bootstrapPluginConfig *PluginConf
	unmountPluginConfig   *PluginConf
	checkPluginConfig     *PluginConf
	result                *types.Result
}

// 以下是 CNIManager 的各种方法，包括获取和设置 CNI 插件、模式、参数、配置和结果等。
func (manager *CNIManager) getCNI(mode string) CNI {
	if cni, ok := manager.cniMap[mode]; ok {
		return cni
	}
	return nil
}

func (manager *CNIManager) getBootstrapMode() string {
	return manager.bootstrapMode
}

func (manager *CNIManager) getUnmountMode() string {
	return manager.unmountMode
}

func (manager *CNIManager) getCheckMode() string {
	return manager.checkMode
}

func (manager *CNIManager) getBootstrapArgs() *skel.CmdArgs {
	return manager.bootstrapArgs
}

func (manager *CNIManager) getUnmountArgs() *skel.CmdArgs {
	return manager.unmountArgs
}

func (manager *CNIManager) getCheckArgs() *skel.CmdArgs {
	return manager.checkArgs
}

func (manager *CNIManager) getBootstrapConfigs() *PluginConf {
	return manager.bootstrapPluginConfig
}

func (manager *CNIManager) getUnmountConfigs() *PluginConf {
	return manager.unmountPluginConfig
}

func (manager *CNIManager) getCheckConfigs() *PluginConf {
	return manager.checkPluginConfig
}

func (manager *CNIManager) getResult() *types.Result {
	return manager.result
}

func (manager *CNIManager) Register(cni CNI) error {
	mode := cni.GetMode()
	if mode == "" {
		return errors.New("插件类型不能为空")
	}
	_cni := manager.getCNI(mode)
	if _cni != nil {
		return errors.New("该类型插件已经存在")
	}
	manager.cniMap[mode] = cni
	return nil
}

func (manager *CNIManager) SetBootstrapConfigs(pluginConfig *PluginConf) *CNIManager {
	manager.bootstrapPluginConfig = pluginConfig
	return manager
}

func (manager *CNIManager) SetUnmountConfigs(pluginConfig *PluginConf) *CNIManager {
	manager.unmountPluginConfig = pluginConfig
	return manager
}

func (manager *CNIManager) SetCheckConfigs(pluginConfig *PluginConf) *CNIManager {
	manager.checkPluginConfig = pluginConfig
	return manager
}

func (manager *CNIManager) SetBootstrapArgs(args *skel.CmdArgs) *CNIManager {
	manager.bootstrapArgs = args
	return manager
}

func (manager *CNIManager) SetUnmountArgs(args *skel.CmdArgs) *CNIManager {
	manager.unmountArgs = args
	return manager
}

func (manager *CNIManager) SetCheckArgs(args *skel.CmdArgs) *CNIManager {
	manager.checkArgs = args
	return manager
}

func (manager *CNIManager) SetBootstrapCNIMode(mode string) *CNIManager {
	manager.bootstrapMode = mode
	return manager
}

func (manager *CNIManager) SetUnmountCNIMode(mode string) *CNIManager {
	manager.unmountMode = mode
	return manager
}

func (manager *CNIManager) SetCheckCNIMode(mode string) *CNIManager {
	manager.checkMode = mode
	return manager
}

// BootstrapCNI 方法用于初始化并启动 CNI 插件。
// 它需要先设置 CNI 插件的 mode（类型）、args（参数）和 configs（配置信息）。
// 如果所需的 CNI 插件未找到，它将返回一个错误。
// 如果在执行 CNI 插件的过程中出现错误，它将记录错误位置并返回错误信息。
func (manager *CNIManager) BootstrapCNI() error {
	mode := manager.getBootstrapMode()
	args := manager.getBootstrapArgs()
	configs := manager.getBootstrapConfigs()
	if mode == "" || args == nil || configs == nil {
		return errors.New("启动 cni 需要设置 mode 和 args 以及 configs")
	}
	cni := manager.getCNI(mode)
	if cni == nil {
		errMsg := fmt.Sprintf("未找到 %s 类型的 cni", mode)
		return errors.New(errMsg)
	}
	cniRes, err := cni.Bootstrap(args, configs)
	if err != nil {
		utils.WriteLog("出错的位置在 cni.Bootstrap")
		return err
	}

	manager.result = cniRes
	return nil
}

// UnmountCNI 方法用于卸载 CNI 插件。
// 它需要先设置 CNI 插件的 mode（类型）、args（参数）和 configs（配置信息）。
// 如果 CNI 插件尚未初始化，它将返回一个错误。
func (manager *CNIManager) UnmountCNI() error {
	mode := manager.getUnmountMode()
	args := manager.getUnmountArgs()
	configs := manager.getUnmountConfigs()
	if mode == "" || args == nil || configs == nil {
		return errors.New("卸载 cni 需要设置 mode 和 args 以及 configs")
	}
	cni := manager.getCNI(mode)
	if cni == nil {
		return errors.New("cni 插件还未初始化, 无法卸载")
	}
	return cni.Unmount(args, configs)
}

// CheckCNI 方法用于检查 CNI 插件的运行状态。
// 它需要先设置 CNI 插件的 mode（类型）、args（参数）和 configs（配置信息）。
// 如果 CNI 插件尚未初始化，它将返回一个错误。
func (manager *CNIManager) CheckCNI() error {
	mode := manager.getCheckMode()
	args := manager.getCheckArgs()
	configs := manager.getCheckConfigs()
	if mode == "" || args == nil || configs == nil {
		return errors.New("检查 cni 需要设置 mode 和 args 以及 configs")
	}
	cni := manager.getCNI(mode)
	if cni == nil {
		return errors.New("cni 插件还未初始化, 无法检查")
	}
	return cni.Check(args, configs)
}

// PrintResult 方法用于打印 CNI 插件的执行结果。
// 如果无法获取到 CNI 插件的执行结果、配置信息或版本信息，它将返回相应的错误信息。
func (manager *CNIManager) PrintResult() error {
	result := manager.getResult()
	if result == nil {
		return errors.New("PrintResult 无法获取到 cni 插件的执行结果")
	}
	config := manager.getBootstrapConfigs()
	if config == nil {
		return errors.New("PrintResult 无法获取到 cni 插件的配置信息")
	}
	version := config.CNIVersion
	if version == "" {
		return errors.New("PrintResult 无法获取到 cni 插件的版本信息")
	}
	cniTypes.PrintResult(result, version)
	return nil
}

// GetCNIManager 函数返回 CNIManager 实例。
func GetCNIManager() *CNIManager {
	return manager
}

// init 函数初始化 CNIManager 实例。
func init() {
	manager = &CNIManager{
		cniMap: map[string]CNI{},
	}
}
