package vxlan

import (
	"cni-demo/cni"
	"cni-demo/consts"
	_etcd "cni-demo/etcd"
	"cni-demo/ipam"
	_ipam "cni-demo/ipam"
	"cni-demo/nettools"
	bpf_map "cni-demo/plugins/vxlan/map"
	"cni-demo/plugins/vxlan/tc"
	"cni-demo/plugins/vxlan/watcher"
	"cni-demo/tools/skel"
	utils2 "cni-demo/tools/utils"
	"errors"
	"fmt"
	types "github.com/containernetworking/cni/pkg/types/100"
	"net"
	"strconv"
	// "github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
)

const MODE = consts.MODE_VXLAN

type VxlanCNI struct {
}

func (vx *VxlanCNI) GetMode() string {
	return MODE
}

// startWatchNodeChange 函数用于启动监听节点变化的进程。如果默认端口已经被占用，说明已经有子进程在监听 etcd 中节点上的 pod ip 变化，此时可以直接跳过。
func startWatchNodeChange(ipam *_ipam.IpamService, etcd *_etcd.EtcdClient) error {
	// 如果这个默认端口已经正在使用了, 则认为之前已经有 pod 在在调用 cni 时启动过监听进程了, 这里可直接跳过
	pidInt, pidStr, err := utils2.GetPidByPort(consts.DEFAULT_TMP_PORT)
	if err == nil && pidInt != -1 {
		// 说明 3190 这个端口正在被监听着, 就跳过 start watcher
		// 先看这个路径是否存在
		if utils2.PathExists(consts.KUBE_TEST_CNI_TMP_DEAMON_DEFAULT_PATH) {
			// 如果该路径存在
			prevPidStr, err := utils2.ReadContentFromFile(consts.KUBE_TEST_CNI_TMP_DEAMON_DEFAULT_PATH)
			// 尝试读出里头的 pid, 然后看这个 pid 和当前正在运行中的 port 对应的 pid 是否相等
			if err != nil || prevPidStr != pidStr {
				// 如果这个文件有问题的话, 就删掉文件重新把 pid 存进入
				utils2.DeleteFile(consts.KUBE_TEST_CNI_TMP_DEAMON_DEFAULT_PATH)
				utils2.CreateFile(consts.KUBE_TEST_CNI_TMP_DEAMON_DEFAULT_PATH, ([]byte)(pidStr), 0766)
			}
			return nil
		}
		// 如果该路径不存在文件的话, 可能是谁误删了, 那就再创建一份儿
		utils2.CreateFile(consts.KUBE_TEST_CNI_TMP_DEAMON_DEFAULT_PATH, ([]byte)(pidStr), 0766)
		return nil
	}
	// 走到这里说明还没有一条子进程能监听 etcd 中 node 上的 pod ip 的变换
	// 这里就启动监听
	return watcher.StartMapWatcher(ipam, etcd)
}

// initEveryClient 函数用于初始化 CNI 需要的每个客户端，包括 IPAM、etcd 和 ebpf map。
func initEveryClient(args *skel.CmdArgs, pluginConfig *cni.PluginConf) (*_ipam.IpamService, *_etcd.EtcdClient, *bpf_map.MapsManager, error) {
	_ipam.Init(pluginConfig.Subnet, &ipam.IPAMOptions{
		MaskSegment:      "16",
		PodIpMaskSegment: "32",
	})
	ipam, err := _ipam.GetIpamService()
	if err != nil {
		return nil, nil, nil, errors.New(fmt.Sprintf("初始化 ipam 客户端失败: %s", err.Error()))
	}
	_etcd.Init()
	etcd, err := _etcd.GetEtcdClient()
	if err != nil {
		return nil, nil, nil, errors.New(fmt.Sprintf("初始化 etcd 客户端失败: %s", err.Error()))
	}

	bpfmap, err := bpf_map.GetMapsManager()
	if err != nil {
		return nil, nil, nil, errors.New(fmt.Sprintf("初始化 ebpf map 失败: %s", err.Error()))
	}
	return ipam, etcd, bpfmap, nil
}

// createHostVethPair 函数用于创建主机上的 veth 对。
func createHostVethPair(args *skel.CmdArgs, pluginConfig *cni.PluginConf) (*netlink.Veth, *netlink.Veth, error) {
	hostVeth, _ := netlink.LinkByName("veth_host")
	netVeth, _ := netlink.LinkByName("veth_net")

	if hostVeth != nil && netVeth != nil {
		// 如果已经有了就直接跳过
		return hostVeth.(*netlink.Veth), netVeth.(*netlink.Veth), nil
	}
	return nettools.CreateVethPair("veth_host", 1500, "veth_net")
}

// setUpHostVethPair 函数用于设置主机上的 veth 对。
func setUpHostVethPair(veth ...*netlink.Veth) error {
	for _, v := range veth {
		err := setUpVeth(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// createNsVethPair 函数用于创建网络命名空间内的 veth 对。
func createNsVethPair(args *skel.CmdArgs, pluginConfig *cni.PluginConf) (*netlink.Veth, *netlink.Veth, error) {
	// mtu 表示最大 mac 帧的长度
	// 默认是 1500
	// 因为一个 vxlan 的帧 = max(14) + ip(20) + udp(8) + vxlan 头部(8) + 原始报文
	// 所以一个 vxlan 的外层多了 14 + 20 + 8 + 8 = 50 字节的一个包装
	// 而 vxlan 设备在解封装的时候要求帧长度不能超过 1500
	// 如果按照默认的话现在就是 1550 了
	// 所以这里设置网卡的 mtu 最大是 1450, 也就是原始报文的部分最大是 1450
	mtu := 1450
	ifName := args.IfName
	random := strconv.Itoa(utils2.GetRandomNumber(100000))
	hostName := "ding_lxc_" + random
	return nettools.CreateVethPair(ifName, mtu, hostName)
}

// setIpIntoHostPair 函数用于将 IP 地址设置到主机 veth 对中。
func setIpIntoHostPair(ipam *_ipam.IpamService, veth *netlink.Veth) (string, error) {
	ipexist, _ := nettools.DeviceExistIp(veth)
	if ipexist != "" {
		return ipexist, nil
	}
	// 获取网关地址, 一般就是当前节点所在网段的第一个 ip
	gw, err := ipam.Get().Gateway()
	if err != nil {
		return "", err
	}
	gw = fmt.Sprintf("%s/%s", gw, "32")
	return gw, nettools.SetIpForVxlan(veth.Name, gw)
}

// getNetns 函数用于获取网络命名空间。
func getNetns(_ns string) (*ns.NetNS, error) {
	netns, err := ns.GetNS(_ns)
	if err != nil {
		utils2.WriteLog("获取 ns 失败: ", err.Error())
		return nil, err
	}
	return &netns, nil
}

// setHostVethIntoHost 函数用于将主机 veth 放入主机网络命名空间。
func setHostVethIntoHost(ipam *_ipam.IpamService, veth *netlink.Veth, netns ns.NetNS) error {
	// 把随机起名的 veth 那头放在主机上
	err := nettools.SetVethNsFd(veth, netns)
	if err != nil {
		utils2.WriteLog("把 veth 设置到 host 上失败: ", err.Error())
		return err
	}
	return nil
}

// setIpIntoNsPair 函数用于将 IP 地址设置到网络命名空间的 veth 对中。
func setIpIntoNsPair(ipam *_ipam.IpamService, veth *netlink.Veth) (string, error) {
	// 从 ipam 中拿到一个未使用的 ip 地址
	podIP, err := ipam.Get().UnusedIP()
	if err != nil {
		utils2.WriteLog("获取 podIP 出错, err: ", err.Error())
		return "", err
	}
	podIP = fmt.Sprintf("%s/%s", podIP, "32")
	err = nettools.SetIpForVxlan(veth.Name, podIP)
	if err != nil {
		utils2.WriteLog("给 ns veth 设置 ip 失败, err: ", err.Error())
		return "", err
	}
	return podIP, nil
}

// setUpVeth 函数用于设置 veth 设备。
func setUpVeth(veth *netlink.Veth) error {
	return nettools.SetUpVeth(veth)
}

// setFibTalbeIntoNs 函数用于在网络命名空间中设置 FIB 表。
func setFibTalbeIntoNs(gw string, veth *netlink.Veth) error {
	// 启动之后给这个 netns 设置默认路由 以便让其他网段的包也能从 veth 走到网桥
	gwIp, gwNet, err := net.ParseCIDR(gw)
	if err != nil {
		utils2.WriteLog("创建交换路由失败, err:", err.Error())
		return err
	}
	defIp, defNet, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		utils2.WriteLog("创建交换路由失败, err:", err.Error())
		return err
	}

	// 设置交换路由, 让流量能从路由表中查询到下一条
	// 注意这里在设置交换路由的时候, 第一条的 gw -> 0.0.0.0 需要是 scope_link
	err = nettools.AddRoute(gwNet, defIp, veth, netlink.SCOPE_LINK)
	if err != nil {
		utils2.WriteLog("设置交换路由 gw -> default 失败: ", err.Error())
		return err
	}
	// 然后创建默认的 0.0.0.0 -> gw 时就可以走默认的 scope universe 了
	// 否则会创建失败
	err = nettools.AddRoute(defNet, gwIp, veth)
	if err != nil {
		utils2.WriteLog("设置交换路由 default -> gw 失败: ", err.Error())
		return err
	}
	return nil
}

// setArp 函数用于在给定的网络命名空间和设备上设置 ARP 表项。
func setArp(ipam *_ipam.IpamService, hostns ns.NetNS, veth *netlink.Veth, dev string) error {
	gw, err := ipam.Get().Gateway()
	if err != nil {
		return err
	}

	err = hostns.Do(func(nn ns.NetNS) error {
		// 这里需要重新获取, 因为上边把这个 veth 从 ns 中给挪到了 host 上
		// 导致 mac 发生了变化
		v, err := netlink.LinkByName(veth.Attrs().Name)
		if err != nil {
			return err
		}
		veth = v.(*netlink.Veth)
		mac := veth.LinkAttrs.HardwareAddr
		_mac := mac.String()
		return nn.Do(func(hostns ns.NetNS) error {
			return nettools.CreateArpEntry(gw, _mac, dev)
		})
	})
	return err
}

// setUpHostPair 函数用于设置主机 veth 对。
func setUpHostPair(hostns ns.NetNS, veth *netlink.Veth) error {
	return hostns.Do(func(nn ns.NetNS) error {
		v, err := netlink.LinkByName(veth.Attrs().Name)
		if err != nil {
			return err
		}
		veth = v.(*netlink.Veth)
		return setUpVeth(veth)
	})
}

// setVxlanInfoToLocalMap 函数用于将 Vxlan 信息存储到本地 Map 中。
func setVxlanInfoToLocalMap(bpfmap *bpf_map.MapsManager, vxlan *netlink.Vxlan) error {
	_, err := bpfmap.CreateNodeLocalMap()
	if err != nil {
		return err
	}
	return bpfmap.SetNodeLocalMap(
		bpf_map.LocalNodeMapKey{
			Type: bpf_map.VXLAN_DEV,
		},
		bpf_map.LocalNodeMapValue{
			IfIndex: uint32(vxlan.Attrs().Index),
		},
	)
}

// stuff8Byte 函数将一个字节切片填充为一个 8 字节的数组。
func stuff8Byte(b []byte) [8]byte {
	var res [8]byte
	if len(b) > 8 {
		b = b[0:9]
	}

	for index, _byte := range b {
		res[index] = _byte
	}
	return res
}

// 该函数用于将主机和容器命名空间中的 Veth 对信息存储到 Lxc Map 中。
// 它首先获取主机命名空间中的 hostVeth 设备，然后解析 podIP 并将其转换为 uint32 格式。
// 接着获取 Veth 设备的索引和 MAC 地址，并将这些信息存储到 Lxc Map 中。
func setVethPairInfoToLxcMap(bpfmap *bpf_map.MapsManager, hostNs ns.NetNS, podIP string, hostVeth, nsVeth *netlink.Veth) error {
	err := hostNs.Do(func(nn ns.NetNS) error {
		v, err := netlink.LinkByName(hostVeth.Attrs().Name)
		if err != nil {
			return err
		}
		hostVeth = v.(*netlink.Veth)
		return nil
	})
	if err != nil {
		return err
	}
	netip, _, err := net.ParseCIDR(podIP)
	if err != nil {
		return err
	}
	podIP = netip.String()

	nsVethPodIp := utils2.InetIpToUInt32(podIP)
	hostVethIndex := uint32(hostVeth.Attrs().Index)
	hostVethMac := stuff8Byte(([]byte)(hostVeth.Attrs().HardwareAddr))
	nsVethIndex := uint32(nsVeth.Attrs().Index)
	nsVethMac := stuff8Byte(([]byte)(nsVeth.Attrs().HardwareAddr))

	_, err = bpfmap.CreateLxcMap()
	if err != nil {
		return err
	}
	return bpfmap.SetLxcMap(
		bpf_map.EndpointMapKey{IP: nsVethPodIp},
		bpf_map.EndpointMapInfo{
			IfIndex:    nsVethIndex,
			LxcIfIndex: hostVethIndex,
			MAC:        nsVethMac,
			NodeMAC:    hostVethMac,
		},
	)
}

// 该函数用于将 TC-BPF 附加到 Veth 设备上。它获取 Veth 设备的名称，
// 然后使用 tc.TryAttachBPF 方法附加 TC-BPF 到 Veth 设备的 Ingress 方向。
func attachTcBPFIntoVeth(veth *netlink.Veth) error {
	name := veth.Attrs().Name
	vethIngressBPFPath := tc.GetVethIngressPath()
	return tc.TryAttachBPF(name, tc.INGRESS, vethIngressBPFPath)
}

// 该函数用于创建一个 Vxlan 设备并启动它。它调用 nettools.CreateVxlanAndUp2
// 方法创建并启动一个名为 name 的 Vxlan 设备，MTU 为 1500。
func createVxlan(name string) (*netlink.Vxlan, error) {
	// return nettools.CreateVxlanAndUp(name, 1500)
	return nettools.CreateVxlanAndUp2(name, 1500)
}

// 该函数用于将 TC-BPF 附加到 Vxlan 设备上。它首先获取 Vxlan 设备的名称，
// 然后调用 tc.TryAttachBPF 方法附加 TC-BPF 到 Vxlan 设备的 Ingress 方向。
// 接着，获取 Vxlan 设备的 Egress 方向 BPF 路径，
// 并调用 tc.TryAttachBPF 方法附加 TC-BPF 到 Vxlan 设备的 Egress 方向。
func attachTcBPFIntoVxlan(vxlan *netlink.Vxlan) error {
	name := vxlan.Attrs().Name
	vxlanIngressBPFPath := tc.GetVxlanIngressPath()
	err := tc.TryAttachBPF(name, tc.INGRESS, vxlanIngressBPFPath)
	if err != nil {
		return err
	}
	vxlanEgressBPFPath := tc.GetVxlanEgressPath()
	return tc.TryAttachBPF(name, tc.EGRESS, vxlanEgressBPFPath)
}

/**
* 该函数是 Vxlan 模式 CNI 插件的主要入口。它首先初始化 IPAM、etcd 和 bpfmap 客户端。
* 然后开始监听 etcd 中 pod 和 subnet map 的变化，并在主机上创建一对 Veth Pair 设备作为默认网关。
* 接下来，将一对 Veth 设备加入到容器命名空间，并为容器命名空间中的 Veth 设备分配 IP 地址。
* 最后，为 Vxlan 设备附加 TC-BPF 并将设备信息存储到本地 Map 中。
* pluginConfig:
 * {
 *   "cniVersion": "0.3.0",
 *   "name": "cni-demo",
 *   "type": "cni-demo",
 *   "mode": "vxlan",
 *   "subnet": "10.244.0.0"
 * }
*/
/**
 * tc qdisc add dev ${pod veth name} clsact
 * tc qdisc add dev ding_vxlan clsact
 * clang -g  -O2 -emit-llvm -c vxlan_egress.c -o - | llc -march=bpf -filetype=obj -o vxlan_egress.o
 * clang -g  -O2 -emit-llvm -c vxlan_ingress.c -o - | llc -march=bpf -filetype=obj -o vxlan_ingress.o
 * clang -g  -O2 -emit-llvm -c veth_ingress.c -o - | llc -march=bpf -filetype=obj -o veth_ingress.o
 * tc filter add dev ding_vxlan egress bpf direct-action obj vxlan_egress.o
 * tc filter add dev ding_vxlan ingress bpf direct-action obj vxlan_ingress.o
 * tc filter add dev ${pod veth name} ingress bpf direct-action obj veth_ingress.o
 */
func (vx *VxlanCNI) Bootstrap(args *skel.CmdArgs, pluginConfig *cni.PluginConf) (*types.Result, error) {
	utils2.WriteLog("进到了 vxlan 模式了")

	// 0. 先把各种能用的上的客户端初始化咯
	ipam, etcd, bpfmap, err := initEveryClient(args, pluginConfig)
	if err != nil {
		return nil, err
	}

	// 1. 开始监听 etcd 中 pod 和 subnet map 的变化, 注意该行为只能有一次
	err = startWatchNodeChange(ipam, etcd)
	if err != nil {
		return nil, err
	}

	// 2. 创建一对 veth pair 设备 veth_host 和 veth_net 作为默认网关
	gwPair, netPair, err := createHostVethPair(args, pluginConfig)
	if err != nil {
		return nil, err
	}

	// 启动这俩设备
	err = setUpHostVethPair(gwPair, netPair)
	if err != nil {
		return nil, err
	}

	// 3. 给这对儿网关 veth 设备中的 veth_host 加上 ip/32
	gw, err := setIpIntoHostPair(ipam, gwPair)
	if err != nil {
		return nil, err
	}

	// 4. 获取 ns
	netns, err := getNetns(args.Netns)
	if err != nil {
		return nil, err
	}

	var nsPair, hostPair *netlink.Veth
	var podIP string
	err = (*netns).Do(func(hostNs ns.NetNS) error {
		// 5. 创建一对儿 veth pair 作为 pod 的 veth
		nsPair, hostPair, err = createNsVethPair(args, pluginConfig)
		if err != nil {
			return err
		}
		// 6. 将 veth pair 设备加入到 kubelet 传来的 ns 下
		err = setHostVethIntoHost(ipam, hostPair, hostNs)
		if err != nil {
			return err
		}

		// 7. 给 ns 中的 veth 创建 ip/32, etcd 会自动通知其他 node
		podIP, err = setIpIntoNsPair(ipam, nsPair)
		if err != nil {
			return err
		}

		// 启动 ns pair
		err = setUpVeth(nsPair)
		if err != nil {
			return err
		}

		// 8. 给这个 ns 中创建默认的路由表以及 arp 表, 让其能把流量都走到 ns 外
		err = setFibTalbeIntoNs(gw, nsPair)
		if err != nil {
			return err
		}

		err = setArp(ipam, hostNs, hostPair, args.IfName)
		if err != nil {
			return err
		}

		// 启动 ns 留在 host 上那半拉 veth
		err = setUpHostPair(hostNs, hostPair)
		if err != nil {
			return err
		}

		// 9. 将 veth pair 的信息写入到 LXC_MAP_DEFAULT_PATH
		err = setVethPairInfoToLxcMap(bpfmap, hostNs, podIP, hostPair, nsPair)
		if err != nil {
			return err
		}
		// TODO(这步暂时不要好像也 ok): 10. 将 veth pair 的 ip 与 node ip 的映射写入到 NODE_LOCAL_MAP_DEFAULT_PATH
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 11. 给 veth pair 中留在 host 上的那半拉的 tc 打上 ingress
	err = attachTcBPFIntoVeth(hostPair)
	if err != nil {
		return nil, err
	}

	// 12. 创建一块儿 vxlan 设备
	vxlan, err := createVxlan("ding_vxlan")
	if err != nil {
		return nil, err
	}

	// 13. 把 vxlan 加入到 NODE_LOCAL_MAP_DEFAULT_PATH
	err = setVxlanInfoToLocalMap(bpfmap, vxlan)
	if err != nil {
		return nil, err
	}

	// 14. 给这块儿 vxlan 设备的 tc 打上 ingress 和 egress
	err = attachTcBPFIntoVxlan(vxlan)
	if err != nil {
		return nil, err
	}

	// 最后交给外头去打印到标准输出
	_gw, _, _ := net.ParseCIDR(gw)
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

// 该函数用于卸载 Vxlan 模式 CNI 插件。目前尚未实现。
func (hostGW *VxlanCNI) Unmount(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) error {
	// TODO
	return nil
}

// 该函数用于检查 Vxlan 模式 CNI 插件的状态。目前尚未实现。
func (hostGW *VxlanCNI) Check(
	args *skel.CmdArgs,
	pluginConfig *cni.PluginConf,
) error {
	// TODO
	return nil
}

// 该函数用于初始化并注册 Vxlan 模式 CNI 插件。它首先创建一个 VxlanCNI 实例，
// 然后将其注册到 CNI Manager 中。如果注册成功，输出成功信息；否则，输出错误信息并触发 panic。
func init() {
	VxlanCNI := &VxlanCNI{}
	manager := cni.GetCNIManager()
	err := manager.Register(VxlanCNI)
	utils2.WriteLog("即将注册 vxlan 模式 cni")
	if err != nil {
		utils2.WriteLog("注册 vxlan cni 失败: ", err.Error())
		panic(err.Error())
	}
	utils2.WriteLog("注册 vxlan 模式 cni 成功")
}
