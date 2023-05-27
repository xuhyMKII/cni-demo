#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/if_arp.h>
#include <linux/if_ether.h>
#include <netinet/in.h>

#include "common.h"
#include "maps.h"

/**
 * 此 eBPF 程序的主要目的是处理从 VXLAN 设备收到的数据包，并将其发送到其他节点上不同网段的 Pod。
 * 程序首先检查数据包的协议类型是否为 IP 协议，然后获取源 IP 和目标 IP 地址。接下来，
 * 它尝试在 eBPF map (ding_ip) 中查找目标 IP 地址所在的节点 IP。如果查找成功，
 * 程序将为数据包设置一个隧道，并使用 bpf_skb_set_tunnel_key 函数为数据
 * 包设置外部 UDP 隧道目标 IP。隧道键中包含远程节点 IP、隧道 ID、隧道 TOS 和隧道 TTL。
 * 如果 bpf_skb_set_tunnel_key 函数调用成功，程序将返回 TC_ACT_OK，
 * 表示数据包已被正确处理并准备好发送到目标节点。否则，程序将打印一条错误消息，并使用 TC_ACT_SHOT 丢弃数据包。
 * 这个 eBPF 程序主要负责处理跨节点之间的数据包转发，使数据包能够在集群内的各个节点之间正确路由，
 * 以便将流量发送到正确的目标 Pod。通过使用 VXLAN 隧道，程序确保了数据包在跨越不同网络环境时能够正确传输。
 *
 * 如果 vxlan 设备收到了数据包
 * 说明是要发送到其他 node 中不同网段的 pod 上
 * 1. 在 POD_MAP_DEFAULT_PATH 中查询目标 pod 所在的 node ip
 * 2. 用 bpf_skb_set_tunnel_key 给原始数据包设置外层的 udp 的 target ip
 * 
 */

// 定义 eBPF 程序的入口点，作为一个分类器
__section("classifier")
int cls_main(struct __sk_buff *skb) {
  // 一些基本的数据和边界检查
	void *data = (void *)(long)skb->data;
	void *data_end = (void *)(long)skb->data_end;
	if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) > data_end) {
    return TC_ACT_UNSPEC;
  }
  // 定义并获取以太网头和 IP 头的指针
	struct ethhdr  *eth  = data;
	struct iphdr   *ip   = (data + sizeof(struct ethhdr));
  // 检查协议类型是否为 IP 协议
  if (eth->h_proto != __constant_htons(ETH_P_IP)) {
		return TC_ACT_UNSPEC;
  }

  // 将 IP 地址从主机字节序转换为网络字节序
  __u32 src_ip = htonl(ip->saddr);
  __u32 dst_ip = htonl(ip->daddr);
  // 查询目标 IP 所在的节点 IP
  struct podNodeKey podNodeKey = {};
  podNodeKey.ip = dst_ip;
  struct podNodeValue *podNode = bpf_map_lookup_elem(&ding_ip, &podNodeKey);
  if (podNode) {
    __u32 dst_node_ip = podNode->ip;
    // 准备一个 tunnel
    struct bpf_tunnel_key key;
    int ret;
    __builtin_memset(&key, 0x0, sizeof(key));
    key.remote_ipv4 = podNode->ip;
    key.tunnel_id = DEFAULT_TUNNEL_ID;
    key.tunnel_tos = 0;
    key.tunnel_ttl = 64;
    // 添加外头的隧道 udp
    ret = bpf_skb_set_tunnel_key(skb, &key, sizeof(key), BPF_F_ZERO_CSUM_TX);
    if (ret < 0) {
      bpf_printk("bpf_skb_set_tunnel_key failed");
      return TC_ACT_SHOT;
    }
    return TC_ACT_OK;
  }
  return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
