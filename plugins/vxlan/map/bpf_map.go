package bpf_map

import (
	utils2 "cni-demo/tools/utils"
	"github.com/cilium/ebpf"
)

// func DelAllKeys(m *ebpf.Map) (int, error) {
// 	itor := m.Iterate()
// 	keys := []interface{}{}
// 	// values := []interface{}{}

// 	var key interface{}
// 	var value interface{}
// 	for itor.Next(&key, &value) {
// 		keys = append(keys, key)
// 		// values = append(values, value)
// 	}
// 	return BatchDelKey(m, keys)
// }

// DelKey 函数用于从给定的 eBPF Map 中删除指定的 key。
func DelKey(m *ebpf.Map, key interface{}) error {
	return m.Delete(key)
}

// BatchDelKey 函数用于从给定的 eBPF Map 中批量删除一组 key。
func BatchDelKey(m *ebpf.Map, keys interface{}) (int, error) {
	return m.BatchDelete(keys, &ebpf.BatchOptions{})
}

// SetMap 函数用于在给定的 eBPF Map 中插入或更新一个键值对。
func SetMap(m *ebpf.Map, key, value interface{}) error {
	// err := m.Put(EndpointKey{IP: 6}, EndpointInfo{
	// 	IfIndex: 2,
	// 	LxcID:   3,
	// 	MAC:     4,
	// 	NodeMAC: 5,
	// })
	return m.Put(key, value)
}

// BatchSetMap 函数用于在给定的 eBPF Map 中批量插入或更新一组键值对。
func BatchSetMap(m *ebpf.Map, keys, values interface{}) (int, error) {
	// err := m.Put(EndpointKey{IP: 6}, EndpointInfo{
	// 	IfIndex: 2,
	// 	LxcID:   3,
	// 	MAC:     4,
	// 	NodeMAC: 5,
	// })
	return m.BatchUpdate(keys, values, &ebpf.BatchOptions{})
}

// GetMapValue 函数用于从给定的 eBPF Map 中查找指定 key 对应的 value。
func GetMapValue(m *ebpf.Map, key, valueOut interface{}) error {
	return m.Lookup(key, valueOut)
}

// GetMapByPinned 函数用于通过固定路径加载一个 eBPF Map。可以选择传入一个或多个 ebpf.LoadPinOptions。
func GetMapByPinned(pinPath string, opts ...*ebpf.LoadPinOptions) *ebpf.Map {
	var options *ebpf.LoadPinOptions
	if len(opts) == 0 {
		options = &ebpf.LoadPinOptions{}
	} else {
		options = opts[0]
	}
	m, err := ebpf.LoadPinnedMap(pinPath, options)
	if err != nil {
		utils2.WriteLog("GetMapByPinned failed: ", err.Error())
	}
	return m
}

// createMap 函数用于创建一个新的 eBPF Map，需要指定 Map 名称、类型、键大小、值大小、最大条目数量和标志。
func createMap(
	name string,
	_type ebpf.MapType,
	keySize uint32,
	valueSize uint32,
	maxEntries uint32,
	flags uint32,
) (*ebpf.Map, error) {
	spec := ebpf.MapSpec{
		Name:       name,
		Type:       ebpf.Hash,
		KeySize:    keySize,
		ValueSize:  valueSize,
		MaxEntries: maxEntries,
		Flags:      flags,
	}
	m, err := ebpf.NewMap(&spec)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// CreateOnceMapWithPin 函数用于在固定路径下创建一个只会创建一次的 eBPF Map。如果固定路径已存在，则直接加载该 Map。否则，将创建一个新的 Map，并将其固定到指定的路径。
// 该方法在同一节点上调用多次但是只会创建一个同名的 map
func CreateOnceMapWithPin(
	pinPath string,
	name string,
	_type ebpf.MapType,
	keySize uint32,
	valueSize uint32,
	maxEntries uint32,
	flags uint32,
) (*ebpf.Map, error) {
	if utils2.PathExists(pinPath) {
		return GetMapByPinned(pinPath), nil
	}
	m, err := createMap(
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
	err = m.Pin(pinPath)
	if err != nil {
		return nil, err
	}
	return m, nil
}
