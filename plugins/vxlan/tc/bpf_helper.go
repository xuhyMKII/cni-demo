package tc

import (
	"cni-demo/consts"
	"errors"
)

type BPF_TC_DIRECT string

const (
	INGRESS BPF_TC_DIRECT = "ingress"
	EGRESS  BPF_TC_DIRECT = "egress"
)

// GetVethIngressPath 函数返回 veth ingress eBPF 程序的默认路径。
func GetVethIngressPath() string {
	return consts.KUBE_TEST_CNI_DEFAULT_PATH + "/veth_ingress.o"
}

// GetVxlanIngressPath 函数返回 vxlan ingress eBPF 程序的默认路径。
func GetVxlanIngressPath() string {
	return consts.KUBE_TEST_CNI_DEFAULT_PATH + "/vxlan_ingress.o"
}

// GetVxlanEgressPath 函数返回 vxlan egress eBPF 程序的默认路径。
func GetVxlanEgressPath() string {
	return consts.KUBE_TEST_CNI_DEFAULT_PATH + "/vxlan_egress.o"
}

// TryAttachBPF 函数尝试将 eBPF 程序附加到指定的网络设备（dev）的 ingress 或 egress 方向（由 direct 参数决定）。
// 如果设备上尚未存在 clsact qdisc，则先添加一个。如果设备上已存在相应方向的 eBPF 程序，则跳过附加操作。
func TryAttachBPF(dev string, direct BPF_TC_DIRECT, program string) error {
	// 如果还没有 clsact 这根儿管子就先尝试 add 一个
	if !ExistClsact(dev) {
		err := AddClsactQdiscIntoDev(dev)
		if err != nil {
			return err
		}
	}

	// 如果当前 dev 上已经有 ingress 或者 egress 就跳过
	switch direct {
	case INGRESS:
		if ExistIngress(dev) {
			return nil
		}
		return AttachIngressBPFIntoDev(dev, program)
	case EGRESS:
		if ExistEgress(dev) {
			return nil
		}
		return AttachEgressBPFIntoDev(dev, program)
	}
	return errors.New("unknow error occurred in TryAttachBPF")
}

// DetachBPF 函数用于从指定网络设备（dev）上删除 clsact qdisc。
func DetachBPF(dev string) error {
	return DelClsactQdiscIntoDev(dev)
}
