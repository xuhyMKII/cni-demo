package bird

import (
	"bytes"
	"cni-demo/ipam"
	"cni-demo/tools/utils"
	"os"
	"strings"
	"text/template"
)

// BgpNeighbor 结构体代表一个 BGP 邻居节点。
type BgpNeighbor struct {
	Name     string // Mesh_192_168_64_16
	IP       string
	Hostname string
}

// BirdConfig 结构体包含了用于生成 BIRD 配置文件所需的信息。
type BirdConfig struct {
	HostIP     string
	HostCIDR   string
	Subnet     string
	VethPrefix string
	Neighbors  []BgpNeighbor
}

// getBirdConfig 函数从 IPAM 服务获取所需的配置信息，并构建 BirdConfig 结构体。
// 返回 BirdConfig 结构体指针，如果有错误发生，返回 error 对象。
func getBirdConfig(is *ipam.IpamService) (*BirdConfig, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	cidr, err := is.Get().CIDR(hostname)
	if err != nil {
		return nil, err
	}

	nodeIP, err := is.Get().NodeIp(hostname)
	if err != nil {
		return nil, err
	}

	subnet, err := is.Get().CurrentSubnet()
	if err != nil {
		return nil, err
	}

	otherIps, err := is.Get().AllOtherHostIP()
	if err != nil {
		return nil, err
	}

	tmp := BirdConfig{
		HostIP:     nodeIP,
		HostCIDR:   cidr,
		Subnet:     subnet,
		VethPrefix: "veth",
	}

	neighs := make([]BgpNeighbor, len(otherIps))
	index := 0
	for k, v := range otherIps {
		name := "Mesh_" + strings.Join(strings.Split(v, "."), "_")
		neighs[index] = BgpNeighbor{
			Name:     name,
			IP:       v,
			Hostname: k,
		}
		index++
	}
	tmp.Neighbors = neighs
	return &tmp, nil
}

// GenConfig 函数根据给定的 IPAM 服务生成 BIRD 配置文件内容。
// 返回生成的 BIRD 配置文件内容，如果有错误发生，返回 error 对象。
func GenConfig(is *ipam.IpamService) (string, error) {
	config, err := getBirdConfig(is)
	if err != nil {
		return "", err
	}
	tpl, err := template.New("cfg").Parse(cfgTpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, config)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GenConfigFile 函数根据给定的 IPAM 服务生成 BIRD 配置文件，并将其写入指定的文件路径。
// 如果有错误发生，返回 error 对象。
func GenConfigFile(is *ipam.IpamService) error {
	config, err := GenConfig(is)
	if err != nil {
		return err
	}
	return utils.CreateFile("/opt/cni-demo/bird.cfg", ([]byte)(config), 0766)
}

// cfgTpl 是用于生成 BIRD 配置文件的模板字符串。
var cfgTpl = `
router id {{.HostIP}};

protocol static {
  route {{.HostCIDR}} blackhole;
}

function calico_aggr() {
  if ( net = {{.HostCIDR}} ) then { accept; }
  if ( net ~ {{.HostCIDR}} ) then { reject; }
}

filter calico_export_to_bgp_peers {
  calico_aggr();
  if ( net ~ {{.Subnet}} ) then {
    accept;
  }
  reject;
}

filter calico_kernel_programming {
  if ( net ~ {{.Subnet}} ) then {
    krt_tunnel = "tunl0";
    accept;
  }
  accept;
}

protocol kernel {
  learn;
  persist;
  scan time 2;
  import all;
  export filter calico_kernel_programming;
  graceful restart;
}

protocol device {
  # debug all;
  scan time 2;
}

protocol direct {
  # debug all;
  # 不包括 veth* 和 kube-ipvs*
  interface -"{{.VethPrefix}}*", -"kube-ipvs*", "*";
}

template bgp bgp_template {
  # debug all;
  description "Connection to BGP peer";
  local as 64512;
  multihop;
  gateway recursive;
  import all; 
  export filter calico_export_to_bgp_peers;
  add paths on;
  graceful restart;
  connect delay time 2;
  connect retry time 5;
  error wait time 5,30;
}

{{range $index, $neigh := .Neighbors}}
protocol bgp {{$neigh.Name}} from bgp_template {
  neighbor {{$neigh.IP}} as 64512;
  source address {{$.HostIP}};
}

{{end}}
`
