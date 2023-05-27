package watcher

import (
	"cni-demo/consts"
	"cni-demo/etcd"
	"cni-demo/ipam"
	utils2 "cni-demo/tools/utils"
	"fmt"
	"strconv"
)

// getAllInitPath 函数用于获取所有节点的初始路径。
func getAllInitPath(ipam *ipam.IpamService) (map[string]string, error) {
	networks, err := ipam.Get().AllHostNetwork()
	if err != nil {
		return nil, err
	}
	maps := map[string]string{}
	for _, network := range networks {
		if network.IsCurrentHost || network.CIDR == "" {
			continue
		}
		ips, err := ipam.Get().RecordByHost(network.Hostname)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			maps[ip] = network.Hostname
		}
	}
	return maps, nil
}

// StartMapWatcher 函数用于启动 MapWatcher，负责监听各个节点的变换并将结果更新到 ebpf 的 map 中。
func StartMapWatcher(ipam *ipam.IpamService, etcd *etcd.EtcdClient) error {
	/**
	 * 这里要负责监听各个节点的变换
	 * 并把得到的结果给塞到 ebpf 的 map 中
	 */

	// 先去获取其他节点所有的 ip 地址
	initMaps, err := getAllInitPath(ipam)
	if err != nil {
		return err
	}
	handlers := &Handlers{
		SubnetRecordHandler: InitRecordSyncProcessor(ipam, initMaps),
	}
	watcher, err := GetWatcher(ipam, etcd, handlers)
	if err != nil {
		return err
	}

	child := utils2.StartDeamon(func() {
		watcher.StartWatch()
		// 在最后启动一个 http 服务作为该子进程的健康检查
		utils2.WriteLog("开始启动健康检查的服务")
		startHealthServer()
	})
	utils2.CreateFile(consts.KUBE_TEST_CNI_TMP_DEAMON_DEFAULT_PATH, ([]byte)(strconv.Itoa(child.Pid)), 0766)
	utils2.WriteLog(fmt.Sprintf("启动的守护进程是: %d", child.Pid))
	return nil
}
