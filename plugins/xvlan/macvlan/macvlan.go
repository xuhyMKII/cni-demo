package macvlan

import (
	"cni-demo/cni"
	"cni-demo/consts"
	base "cni-demo/plugins/xvlan/base"
	"cni-demo/tools/skel"
	"cni-demo/tools/utils"
	types "github.com/containernetworking/cni/pkg/types/100"
	"net"
)

// MODE 常量用于表示当前 CNI 的模式为 MACVLAN。
const MODE = consts.MODE_MACVLAN

// MacVlanCNI 结构体表示一个 MacVlan 类型的 CNI 插件。
type MacVlanCNI struct{}

// Bootstrap 方法用于设置 MacVlanCNI 插件的网络设备，传入 skel.CmdArgs 和 cni.PluginConf，返回类型为 *types.Result 的结果和一个 error。
func (macvlan *MacVlanCNI) Bootstrap(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) (*types.Result, error) {
	podIP, gw, err := base.SetXVlanDevice(base.MODE_MACVlan, args, pluginConfig)
	if err != nil {
		return nil, err
	}
	// 获取网关地址和 podIP 准备返回给外边
	_gw := net.ParseIP(gw)
	_, _podIP, _ := net.ParseCIDR(podIP)
	result := &types.Result{
		CNIVersion: pluginConfig.CNIVersion,
		IPs: []*types.IPConfig{
			{
				Address: *_podIP,
				Gateway: _gw,
			},
		},
	}
	return result, nil
}

// Unmount 方法用于卸载 MacVlanCNI 插件的网络设备，传入 skel.CmdArgs 和 cni.PluginConf，返回一个 error。
func (macvlan *MacVlanCNI) Unmount(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) error {
	// TODO
	return nil
}

// Check 方法用于检查 MacVlanCNI 插件的状态，传入 skel.CmdArgs 和 cni.PluginConf，返回一个 error。
func (macvlan *MacVlanCNI) Check(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) error {
	// TODO
	return nil
}

// GetMode 方法返回当前 CNI 插件的模式（MACVLAN）。
func (macvlan *MacVlanCNI) GetMode() string {
	return MODE
}

// init 函数在 MacVlanCNI 插件初始化时将其注册到 CNI Manager。
func init() {
	MacVlanCNI := &MacVlanCNI{}
	manager := cni.GetCNIManager()
	err := manager.Register(MacVlanCNI)
	if err != nil {
		utils.WriteLog("注册 macvlan cni 失败: ", err.Error())
		panic(err.Error())
	}
}
