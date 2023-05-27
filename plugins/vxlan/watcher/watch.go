package watcher

import (
	"cni-demo/etcd"
	"cni-demo/ipam"
	"cni-demo/tools/utils"
	"encoding/json"
	"os"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
)

// ipam：IpamService 实例，用于 IP 地址管理相关操作
// etcd：EtcdClient 实例，用于访问 etcd 数据
// watcher：etcd Watcher 实例，用于监控 etcd 数据的变化
// subnetRecordHandler：etcd WatchCallback 函数类型，处理 subnet 记录变化时调用的回调函数
// isWatching：布尔值，表示是否正在监控数据
// watchingMap：保存当前正在监控的路径和其状态的映射
// mapsPath：要监控的 maps 路径
type WatcherProcess struct {
	ipam                *ipam.IpamService
	etcd                *etcd.EtcdClient
	watcher             *etcd.Watcher
	subnetRecordHandler etcd.WatchCallback
	isWatching          bool
	watchingMap         map[string]bool
	mapsPath            string
}

// SubnetRecordHandler：etcd WatchCallback 函数类型，处理 subnet 记录变化时调用的回调函数
type Handlers struct {
	// HostnameAndSubnetMapsHandler etcd.WatchCallback
	SubnetRecordHandler etcd.WatchCallback
}

// 对 promise 中的每个路径进行监控，将其添加到 watchingMap 中，并在每次添加后暂停 1 秒
func (wp *WatcherProcess) doWatch(promise []string) {
	for _, path := range promise {
		wp.watcher.Watch(path, wp.subnetRecordHandler)
		wp.watchingMap[path] = true
		time.Sleep(1 * time.Second)
	}
}

// 根据当前正在监控的路径和要监控的路径，返回应该监控的路径列表
// 在这个过程中，会过滤掉当前主机的 hostname，即不监控当前主机的数据
// watching 是监听中的 hostname 和网段的映射, promise 是要希望要被监听的地址
func (wp *WatcherProcess) getShouldWatchPath(watching map[string]bool, promise map[string]string) ([]string, error) {
	res := []string{}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	for _, v := range promise {
		// 不用监听自己这台主机
		if hostname == v {
			continue
		}
		path, err := wp.ipam.Get().RecordPathByHost(v)
		if err != nil {
			return nil, err
		}
		// 看该 ip 当前是否已经被监听
		if watched, ok := watching[path]; ok && watched {
			continue
		}
		res = append(res, path)
	}
	return res, nil
}

// 开始监控数据
// 首先获取所有被分配的网段和对应的 hostname
// 获取应该监控的路径，并调用 doWatch 进行监控
// 同时监控 mapsPath 路径，当监听到变化时，获取新的应该监控的路径并调用 doWatch 进行监控
// 返回取消监控的函数
func (wp *WatcherProcess) StartWatch() (func(), error) {
	if wp.isWatching {
		return wp.CancelWatch, nil
	}
	if len(wp.mapsPath) == 0 {
		return utils.Noop, nil
	}

	// 获取所有被分配出去的网段以及对应的 hostname
	maps, err := wp.ipam.Get().HostSubnetMap()
	if err != nil {
		return utils.Noop, err
	}

	paths, err := wp.getShouldWatchPath(wp.watchingMap, maps)
	if err != nil {
		return utils.Noop, err
	}
	// 开始监听这些路径
	wp.doWatch(paths)

	// 然后再开始监听 hostname 和网段关系映射的地址
	wp.watcher.Watch(wp.mapsPath, func(_type mvccpb.Event_EventType, key, value []byte) {
		// 每次监听到 maps 路径的变化时应该就多监听一个新加进来的 key
		newMaps := map[string]string{}
		err := json.Unmarshal(value, &newMaps)
		if err != nil {
			return
		}
		paths, err := wp.getShouldWatchPath(wp.watchingMap, newMaps)
		if err != nil {
			return
		}
		wp.doWatch(paths)
	})
	return wp.CancelWatch, nil
}

// 取消监控，将 isWatching 设置为 false，并调用 watcher 的 Cancel 方法
func (wp *WatcherProcess) CancelWatch() {
	wp.isWatching = false
	cancel := wp.watcher.Cancel
	cancel()
}

// 初始化 WatcherProcess 实例，设置 ipam、etcd、watchingMap 和 subnetRecordHandler
// 获取 mapsPath，设置到 WatcherProcess 中
// 创建 etcd Watcher 实例，设置到 WatcherProcess 中
var GetWatcher = func() func(ipam *ipam.IpamService, etcd *etcd.EtcdClient, handlers *Handlers) (*WatcherProcess, error) {
	var wp *WatcherProcess
	return func(ipam *ipam.IpamService, etcd *etcd.EtcdClient, handlers *Handlers) (*WatcherProcess, error) {
		if wp != nil {
			return wp, nil
		}
		wp = &WatcherProcess{
			ipam:                ipam,
			etcd:                etcd,
			watchingMap:         map[string]bool{},
			subnetRecordHandler: handlers.SubnetRecordHandler,
		}

		mapsPath, err := ipam.Get().HostSubnetMapPath()
		if err != nil {
			return nil, err
		}
		wp.mapsPath = mapsPath

		watcher, err := etcd.GetWatcher()
		if err != nil {
			return nil, err
		}
		wp.watcher = watcher
		return wp, nil
	}
}()
