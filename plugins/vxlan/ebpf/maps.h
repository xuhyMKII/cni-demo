#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>

#include "common.h"

//这个代码片段定义了三个 eBPF maps，分别是 `ding_lxc`、`ding_ip` 和 `ding_local`。
//它们分别用于存储终端信息、Pod 节点信息以及本地节点信息。这些 maps 的类型都是哈希表，
//且最大条目数为 255。同时，这些 maps 的 pinning 类型都被指定为 `LIBBPF_PIN_BY_NAME`，
//意味着它们将与一个文件系统路径关联。具体的键值类型根据不同的 map 而异。

// 定义本地设备类型：VXLAN 和 VETH
#define LOCAL_DEV_VXLAN 1;
#define LOCAL_DEV_VETH 2;

// 默认的隧道 ID
#define DEFAULT_TUNNEL_ID 13190

// 定义 endpointKey 结构体，用于存储终端 IP 地址
struct endpointKey {
  __u32 ip;
};

// 定义 endpointInfo 结构体，用于存储终端相关信息
struct endpointInfo {
  __u32 ifIndex;      // 接口索引
  __u32 lxcIfIndex;   // 容器网络接口索引
  __u8 mac[8];        // MAC 地址
  __u8 nodeMac[8];    // 节点 MAC 地址
};

// 定义一个名为 ding_lxc 的 eBPF map，用于存储 endpointKey 和 endpointInfo
struct {
	__uint(type, BPF_MAP_TYPE_HASH);          // map 类型为哈希表
  __uint(max_entries, 255);                 // 最大条目数为 255
	__type(key, struct endpointKey);          // 键类型为 endpointKey
  __type(value, struct endpointInfo);       // 值类型为 endpointInfo
  // 如果别的地方已经往某条路径 pin 了, 需要加上这个属性
  // 并且 struct 的名字一定得和 bpftool map list 出来的一样
  __uint(pinning, LIBBPF_PIN_BY_NAME);      // 指定 pinning 类型，将 map 与一个文件系统路径关联
// 加了 SEC(".maps") 的话, clang 在编译时需要加 -g 参数用来生成调试信息
// 这里 ding_lxc 是必须要和 bpftool map list 出来的那个 pinned 中路径的名字一样
} ding_lxc __section_maps_btf;

// 定义 podNodeKey 结构体，用于存储 Pod 节点 IP 地址
struct podNodeKey {
  __u32 ip;
};

// 定义 podNodeValue 结构体，用于存储 Pod 节点相关信息
struct podNodeValue {
  __u32 ip;
};

// 定义一个名为 ding_ip 的 eBPF map，用于存储 podNodeKey 和 podNodeValue
struct {
	__uint(type, BPF_MAP_TYPE_HASH);          // map 类型为哈希表
  __uint(max_entries, 255);                 // 最大条目数为 255
	__type(key, struct podNodeKey);           // 键类型为 podNodeKey
  __type(value, struct podNodeValue);       // 值类型为 podNodeValue
  __uint(pinning, LIBBPF_PIN_BY_NAME);      // 指定 pinning 类型，将 map 与一个文件系统路径关联
} ding_ip __section_maps_btf;

// 定义 localNodeMapKey 结构体，用于存储本地节点类型
struct localNodeMapKey {
	__u32 type;
};

// 定义 localNodeMapValue 结构体，用于存储本地节点相关信息
struct localNodeMapValue {
  __u32 ifIndex;  // 接口索引
};

// 定义一个名为 ding_local 的 eBPF map，用于存储 localNodeMapKey 和 localNodeMapValue
struct {
	__uint(type, BPF_MAP_TYPE_HASH);          // map 类型为哈希表
  __uint(max_entries, 255);                 // 最大条目数为 255
	__type(key, struct localNodeMapKey);      // 键类型为 localNodeMapKey
__type(value, struct localNodeMapValue); // 值类型为 localNodeMapValue
__uint(pinning, LIBBPF_PIN_BY_NAME); // 指定 pinning 类型，将 map 与一个文件系统路径关联
} ding_local __section_maps_btf;

