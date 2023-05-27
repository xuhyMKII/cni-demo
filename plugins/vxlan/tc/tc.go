package tc

import (
	"fmt"
	"os/exec"
	"strings"
)

// _exec 函数是一个辅助函数，用于执行给定的 shell 命令。
// TODO: 可以尝试换成 go-tc
func _exec(command string) error {
	processInfo := exec.Command("/bin/sh", "-c", command)
	_, err := processInfo.Output()
	return err
}

// AddClsactQdiscIntoDev 函数用于为指定的网络设备（dev）添加 clsact qdisc。
func AddClsactQdiscIntoDev(dev string) error {
	return _exec(
		fmt.Sprintf("tc qdisc add dev %s clsact", dev),
	)
}

// DelClsactQdiscIntoDev 函数用于从指定的网络设备（dev）上删除 clsact qdisc。
func DelClsactQdiscIntoDev(dev string) error {
	return _exec(
		fmt.Sprintf("tc qdisc del dev %s clsact", dev),
	)
}

// AttachIngressBPFIntoDev 函数用于将 ingress eBPF 程序附加到指定的网络设备（dev）上。
func AttachIngressBPFIntoDev(dev string, filepath string) error {
	return _exec(
		fmt.Sprintf("tc filter add dev %s ingress bpf direct-action obj %s", dev, filepath),
	)
}

// AttachEgressBPFIntoDev 函数用于将 egress eBPF 程序附加到指定的网络设备（dev）上。
func AttachEgressBPFIntoDev(dev string, filepath string) error {
	return _exec(
		fmt.Sprintf("tc filter add dev %s egress bpf direct-action obj %s", dev, filepath),
	)
}

// ExistClsact 函数检查指定的网络设备（dev）上是否存在 clsact qdisc。
func ExistClsact(dev string) bool {
	processInfo := exec.Command(
		"/bin/sh", "-c",
		fmt.Sprintf("tc qdisc show dev %s", dev),
	)
	out, _ := processInfo.Output()
	return strings.Contains(string(out), "clsact")
}

// ExistIngress 函数检查指定的网络设备（dev）上是否存在 ingress eBPF 程序。
func ExistIngress(dev string) bool {
	processInfo := exec.Command(
		"/bin/sh", "-c",
		fmt.Sprintf("tc filter show dev %s ingress", dev),
	)
	out, _ := processInfo.Output()
	return strings.Contains(string(out), "direct-action")
}

// ExistEgress 函数检查指定的网络设备（dev）上是否存在 egress eBPF 程序。
func ExistEgress(dev string) bool {
	processInfo := exec.Command(
		"/bin/sh", "-c",
		fmt.Sprintf("tc filter show dev %s egress", dev),
	)
	out, _ := processInfo.Output()
	return strings.Contains(string(out), "direct-action")
}

// ShowBPF 函数显示指定网络设备（dev）上指定方向（direct 参数，可为 "ingress" 或 "egress"）的 eBPF 信息。
// 返回 eBPF 信息字符串，如果发生错误则返回错误。
func ShowBPF(dev string, direct string) (string, error) {
	processInfo := exec.Command(
		"/bin/sh", "-c",
		fmt.Sprintf("tc filter show dev %s %s", dev, direct),
	)
	out, err := processInfo.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
