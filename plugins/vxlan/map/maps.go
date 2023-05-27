package bpf_map

import (
	"cni-demo/tools/utils"
	"unsafe"

	"github.com/cilium/ebpf"
)

type MapsManager struct{}

// DeleteAllPodMap 方法用于删除 PodMap 中的所有键值对。
func (mm *MapsManager) DeleteAllPodMap() (int, error) {
	m := mm.GetPodMap()
	itor := m.Iterate()
	keys := []PodNodeMapKey{}

	var key PodNodeMapKey
	var value PodNodeMapValue
	for itor.Next(&key, &value) {
		keys = append(keys, key)
	}
	return BatchDelKey(m, keys)
}

// BatchDelLxcMap 方法用于批量删除 LxcMap 中的一组键。
func (mm *MapsManager) BatchDelLxcMap(keys []EndpointMapKey) (int, error) {
	m := mm.GetLxcMap()
	return BatchDelKey(m, keys)
}

// BatchDelPodMap 方法用于批量删除 PodMap 中的一组键。
func (mm *MapsManager) BatchDelPodMap(keys []PodNodeMapKey) (int, error) {
	m := mm.GetPodMap()
	return BatchDelKey(m, keys)
}

// BatchDelNodeLocalMap 方法用于批量删除 NodeLocalMap 中的一组键。
func (mm *MapsManager) BatchDelNodeLocalMap(keys []LocalNodeMapKey) (int, error) {
	m := mm.GetNodeLocalMap()
	return BatchDelKey(m, keys)
}

// BatchSetLxcMap 方法用于批量设置 LxcMap 中的一组键值对。
func (mm *MapsManager) BatchSetLxcMap(key []EndpointMapKey, value []EndpointMapInfo) (int, error) {
	m := mm.GetLxcMap()
	return BatchSetMap(m, key, value)
}

// BatchSetPodMap 方法用于批量设置 PodMap 中的一组键值对。
func (mm *MapsManager) BatchSetPodMap(key []PodNodeMapKey, value []PodNodeMapValue) (int, error) {
	m := mm.GetPodMap()
	return BatchSetMap(m, key, value)
}

// BatchSetNodeLocalMap 方法用于批量设置 NodeLocalMap 中的一组键值对。
func (mm *MapsManager) BatchSetNodeLocalMap(key []LocalNodeMapKey, value []LocalNodeMapValue) (int, error) {
	m := mm.GetNodeLocalMap()
	return BatchSetMap(m, key, value)
}

// SetLxcMap 方法用于设置 LxcMap 中的一个键值对。
func (mm *MapsManager) SetLxcMap(key EndpointMapKey, value EndpointMapInfo) error {
	m := mm.GetLxcMap()
	return SetMap(m, key, value)
}

// SetPodMap 方法用于设置 PodMap 中的一个键值对。
func (mm *MapsManager) SetPodMap(key PodNodeMapKey, value PodNodeMapValue) error {
	m := mm.GetPodMap()
	return SetMap(m, key, value)
}

// SetNodeLocalMap 方法用于设置 NodeLocalMap 中的一个键值对。
func (mm *MapsManager) SetNodeLocalMap(key LocalNodeMapKey, value LocalNodeMapValue) error {
	m := mm.GetNodeLocalMap()
	return SetMap(m, key, value)
}

// DelLxcMap 方法用于删除 LxcMap 中的一个键。
func (mm *MapsManager) DelLxcMap(key EndpointMapKey) error {
	m := mm.GetLxcMap()
	return DelKey(m, key)
}

// DelPodMap 方法用于删除 PodMap 中的一个键。
func (mm *MapsManager) DelPodMap(key PodNodeMapKey) error {
	m := mm.GetPodMap()
	return DelKey(m, key)
}

// DelNodeLocalMap 方法用于删除 NodeLocalMap 中的一个键。
func (mm *MapsManager) DelNodeLocalMap(key LocalNodeMapKey) error {
	m := mm.GetNodeLocalMap()
	return DelKey(m, key)
}

// GetLxcMap 方法用于通过固定路径加载 LxcMap。
func (mm *MapsManager) GetLxcMap() *ebpf.Map {
	return GetMapByPinned(LXC_MAP_DEFAULT_PATH)
}

// GetPodMap 方法用于通过固定路径加载 PodMap。
func (mm *MapsManager) GetPodMap() *ebpf.Map {
	return GetMapByPinned(POD_MAP_DEFAULT_PATH)
}

// GetNodeLocalMap 方法用于通过固定路径加载 NodeLocalMap。
func (mm *MapsManager) GetNodeLocalMap() *ebpf.Map {
	return GetMapByPinned(NODE_LOCAL_MAP_DEFAULT_PATH)
}

// GetLxcMapValue 方法用于获取 LxcMap 中指定键的值。
func (mm *MapsManager) GetLxcMapValue(key EndpointMapKey) (*EndpointMapInfo, error) {
	m := mm.GetLxcMap()
	value := &EndpointMapInfo{}
	err := GetMapValue(m, key, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// GetPodMapValue 方法用于获取 PodMap 中指定键的值。
func (mm *MapsManager) GetPodMapValue(key PodNodeMapKey) (*PodNodeMapValue, error) {
	m := mm.GetPodMap()
	value := &PodNodeMapValue{}
	err := GetMapValue(m, key, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// GetNodeLocalMapValue 方法用于获取 NodeLocalMap 中指定键的值。
func (mm *MapsManager) GetNodeLocalMapValue(key LocalNodeMapKey) (*LocalNodeMapValue, error) {
	m := mm.GetNodeLocalMap()
	value := &LocalNodeMapValue{}
	err := GetMapValue(m, key, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// CreateLxcMap 方法用于创建一个用于存储本地 veth pair 网卡的 LxcMap。
// 创建一个用来存储本地 veth pair 网卡的 map
func (mm *MapsManager) CreateLxcMap() (*ebpf.Map, error) {
	const (
		pinPath    = LXC_MAP_DEFAULT_PATH
		name       = "lxc_map"
		_type      = ebpf.Hash
		keySize    = uint32(unsafe.Sizeof(EndpointMapKey{}))
		valueSize  = uint32(unsafe.Sizeof(EndpointMapInfo{}))
		maxEntries = MAX_ENTRIES
		flags      = 0
	)

	m, err := CreateOnceMapWithPin(
		pinPath,
		name,
		_type,
		keySize,
		valueSize,
		maxEntries,
		flags,
	)

	if err != nil {
		return nil, err
	}
	return m, nil
}

// CreatePodMap 方法用于创建一个用于存储集群中其他节点上的 Pod IP 的 PodMap。
// 创建一个用来存储集群中其他节点上的 pod ip 的 map
func (mm *MapsManager) CreatePodMap() (*ebpf.Map, error) {
	const (
		pinPath    = POD_MAP_DEFAULT_PATH
		name       = "pod_map"
		_type      = ebpf.Hash
		keySize    = uint32(unsafe.Sizeof(PodNodeMapKey{}))
		valueSize  = uint32(unsafe.Sizeof(PodNodeMapValue{}))
		maxEntries = MAX_ENTRIES
		flags      = 0
	)

	m, err := CreateOnceMapWithPin(
		pinPath,
		name,
		_type,
		keySize,
		valueSize,
		maxEntries,
		flags,
	)

	if err != nil {
		return nil, err
	}
	return m, nil
}

// CreateNodeLocalMap 方法用于创建一个用于存储本机网卡设备的 NodeLocalMap。
// 创建一个用来存储本机网卡设备的 map
func (mm *MapsManager) CreateNodeLocalMap() (*ebpf.Map, error) {
	const (
		pinPath    = NODE_LOCAL_MAP_DEFAULT_PATH
		name       = "local_map"
		_type      = ebpf.Hash
		keySize    = uint32(unsafe.Sizeof(LocalNodeMapKey{}))
		valueSize  = uint32(unsafe.Sizeof(LocalNodeMapValue{}))
		maxEntries = MAX_ENTRIES
		flags      = 0
	)

	m, err := CreateOnceMapWithPin(
		pinPath,
		name,
		_type,
		keySize,
		valueSize,
		maxEntries,
		flags,
	)

	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetMapsManager 闭包函数用于创建或返回一个 MapsManager 实例。
// 在首次调用时，它会创建一个新的 MapsManager 实例，并确保相关目录已创建。
// 在后续调用时，它会返回已创建的 MapsManager 实例。
var GetMapsManager = func() func() (*MapsManager, error) {
	var mm *MapsManager
	return func() (*MapsManager, error) {
		if mm != nil {
			return mm, nil
		} else {
			var err error
			mm = &MapsManager{}
			lxcPath := utils.GetParentDirectory(LXC_MAP_DEFAULT_PATH)
			if !utils.PathExists(lxcPath) {
				err = utils.CreateDir(lxcPath)
			}
			podPath := utils.GetParentDirectory(POD_MAP_DEFAULT_PATH)
			if !utils.PathExists(podPath) {
				err = utils.CreateDir(podPath)
			}
			localPath := utils.GetParentDirectory(NODE_LOCAL_MAP_DEFAULT_PATH)
			if !utils.PathExists(localPath) {
				err = utils.CreateDir(localPath)
			}
			if err != nil {
				return nil, err
			}
			return mm, nil
		}
	}
}()
