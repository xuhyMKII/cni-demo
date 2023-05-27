package ipvlan

import (
	"cni-demo/cni"
	"cni-demo/consts"
	base "cni-demo/plugins/xvlan/base"
	"cni-demo/tools/skel"
	"cni-demo/tools/utils"
	types "github.com/containernetworking/cni/pkg/types/100"
	"net"
)

// MODE 常量用于表示当前 CNI 的模式为 IPVLAN。
const MODE = consts.MODE_IPVLAN

// IPVlanCNI 结构体表示一个 IPVlan 类型的 CNI 插件。
type IPVlanCNI struct{}

// Bootstrap 方法用于设置 IPVlanCNI 插件的网络设备，传入 skel.CmdArgs 和 cni.PluginConf，返回类型为 *types.Result 的结果和一个 error。
func (ipvlan *IPVlanCNI) Bootstrap(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) (*types.Result, error) {
	podIP, gw, err := base.SetXVlanDevice(base.MODE_IPVLAN, args, pluginConfig)
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

// Unmount 方法用于卸载 IPVlanCNI 插件的网络设备，传入 skel.CmdArgs 和 cni.PluginConf，返回一个 error。
func (ipvlan *IPVlanCNI) Unmount(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) error {
	// TODO
	return nil
}

// Check 方法用于检查 IPVlanCNI 插件的状态，传入 skel.CmdArgs 和 cni.PluginConf，返回一个 error。
func (ipvlan *IPVlanCNI) Check(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) error {
	// TODO
	return nil
}

// GetMode 方法返回当前 CNI 插件的模式（IPVLAN）。
func (ipvlan *IPVlanCNI) GetMode() string {
	return MODE
}

// init 函数在 IPVlanCNI 插件初始化时将其注册到 CNI Manager。
func init() {
	IPVlanCNI := &IPVlanCNI{}
	manager := cni.GetCNIManager()
	err := manager.Register(IPVlanCNI)
	if err != nil {
		utils.WriteLog("注册 ipvlan cni 失败: ", err.Error())
		panic(err.Error())
	}
}
