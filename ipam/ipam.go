package ipam

/**
 * 可通过命令查看 etcd 集群状态
 * ETCDCTL_API=3 etcdctl --endpoints https://192.168.98.143:2379 --cacert /etc/kubernetes/pki/etcd/ca.crt --cert /etc/kubernetes/pki/etcd/healthcheck-client.crt --key /etc/kubernetes/pki/etcd/healthcheck-client.key get / --prefix --keys-only
 */

import (
	"cni-demo/client"
	"cni-demo/consts"
	"cni-demo/etcd"
	utils2 "cni-demo/tools/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vishvananda/netlink"
	oriEtcd "go.etcd.io/etcd/client/v3"
	"os"
	"strings"
	"sync"
)

const (
	prefix = "cni-demo/ipam"
)

type Get struct {
	etcdClient *etcd.EtcdClient
	k8sClient  *client.LightK8sClient
	// 有些不会发生改变的东西可以做缓存
	nodeIpCache map[string]string
	cidrCache   map[string]string
}
type Release struct {
	etcdClient *etcd.EtcdClient
	k8sClient  *client.LightK8sClient
}
type Set struct {
	etcdClient *etcd.EtcdClient
	k8sClient  *client.LightK8sClient
}

// operator 结构体用于获取、设置和释放 IP 地址
type operators struct {
	Get     *Get
	Set     *Set
	Release *Release
}

type operator struct {
	*operators
}

type Network struct {
	Name          string
	IP            string
	Hostname      string
	CIDR          string
	IsCurrentHost bool
}

// IpamService 结构体定义了 IPAM 服务的基本信息和属性
type IpamService struct {
	// 子网网络地址
	Subnet string
	// 子网掩码位数
	MaskSegment string
	// 子网掩码 IP 地址
	MaskIP string
	// Pod 子网掩码位数
	PodMaskSegment string
	// Pod 子网掩码 IP 地址
	PodMaskIP string
	// 当前节点分配的网络地址
	CurrentHostNetwork string
	// Etcd 客户端
	EtcdClient *etcd.EtcdClient
	// Kubernetes 客户端
	K8sClient *client.LightK8sClient
	*operator
}

// IPAMOptions 结构体定义了 IPAM 初始化时的配置选项
type IPAMOptions struct {
	// 自定义子网掩码位数
	MaskSegment string
	// 自定义 Pod 子网掩码位数
	PodIpMaskSegment string
	// 自定义 IP 地址范围起始地址
	RangeStart string
	// 自定义 IP 地址范围结束地址
	RangeEnd string
}

// 定义互斥锁及锁状态变量，用于保证同一时间只有一个线程操作 IP 分配
var _lock sync.Mutex
var _isLocking bool

// unlock 函数用于解锁互斥锁
func unlock() {
	if _isLocking {
		_lock.Unlock()
		_isLocking = false
	}
}

// lock 函数用于上锁互斥锁
func lock() {
	if !_isLocking {
		_lock.Lock()
		_isLocking = true
	}
}

// getEtcdClient 函数用于获取 Etcd 客户端实例
func getEtcdClient() *etcd.EtcdClient {
	etcd.Init()
	etcdClient, err := etcd.GetEtcdClient()
	if err != nil {
		return nil
	}
	return etcdClient
}

// getLightK8sClient 函数用于获取 Kubernetes 客户端实例
func getLightK8sClient() *client.LightK8sClient {
	paths, err := client.GetHostAuthenticationInfoPath()
	if err != nil {
		utils2.WriteLog("GetHostAuthenticationInfoPath 执行失败")
		return nil
	}
	client.Init(paths.CaPath, paths.CertPath, paths.KeyPath)
	k8sClient, err := client.GetLightK8sClient()
	if err != nil {
		return nil
	}
	return k8sClient
}

// getIpamSubnet 函数用于获取 IPAM 服务的子网地址
func getIpamSubnet() string {
	ipam, _ := GetIpamService()
	return ipam.Subnet
}

// getIpamMaskSegment 函数用于获取 IPAM 服务的子网掩码位数
func getIpamMaskSegment() string {
	ipam, _ := GetIpamService()
	return ipam.MaskSegment
}

// getHostPath 函数用于获取以主机名为子目录的路径
func getHostPath() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "/test-error-path"
	}
	return getEtcdPathWithPrefix("/" + getIpamSubnet() + "/" + getIpamMaskSegment() + "/" + hostname)
}

// getRecordPath 函数用于获取主机网络记录的路径
func getRecordPath(hostNetwork string) string {
	return getHostPath() + "/" + hostNetwork
}

// getIpRangesPath 函数用于获取 IP 范围路径
func getIpRangesPath(network string) string {
	return getHostPath() + "/" + network + "/range"
}

// getIPsPoolPath 函数用于获取 IP 池路径
func getIPsPoolPath(subnet, mask string) string {
	return getEtcdPathWithPrefix("/" + subnet + "/" + mask + "/" + "pool")
}

// MaskSegment 方法返回 IPAM 服务的子网掩码位数
func (g *Get) MaskSegment() (string, error) {
	ipam, err := GetIpamService()
	if err != nil {
		return "", err
	}
	return ipam.MaskSegment, nil
}

// Subnet 方法返回 IPAM 服务的子网地址
func (g *Get) Subnet() (string, error) {
	ipam, err := GetIpamService()
	if err != nil {
		return "", err
	}
	return ipam.Subnet, nil
}

// HostSubnetMapPath 方法返回主机子网映射的路径
func (g *Get) HostSubnetMapPath() (string, error) {
	ipam, err := GetIpamService()
	if err != nil {
		return "", err
	}
	m := fmt.Sprintf("/%s/%s/maps", ipam.Subnet, ipam.MaskSegment)
	return getEtcdPathWithPrefix(m), nil
}

// HostSubnetMap 方法返回主机子网映射的数据
func (g *Get) HostSubnetMap() (map[string]string, error) {
	ipam, err := GetIpamService()
	if err != nil {
		return nil, err
	}
	return ipam.getHostSubnetMap()
}

// RecordPathByHost 方法根据主机名返回网络记录的路径
func (g *Get) RecordPathByHost(hostname string) (string, error) {
	cidr, err := g.CIDR(hostname)
	if err != nil {
		return "", err
	}
	subnetAndMask := strings.Split(cidr, "/")
	if len(subnetAndMask) > 1 {
		path := fmt.Sprintf("/%s/%s/%s/%s", getIpamSubnet(), getIpamMaskSegment(), hostname, subnetAndMask[0])
		return getEtcdPathWithPrefix(path), nil
	}
	return "", errors.New("can not get subnet address")
}

// CurrentSubnet 方法返回当前子网的 CIDR 格式
func (g *Get) CurrentSubnet() (string, error) {
	ipam, err := GetIpamService()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", ipam.Subnet, ipam.MaskSegment), nil
}

// RecordByHost 方法根据主机名返回网络记录
func (g *Get) RecordByHost(hostname string) ([]string, error) {
	path, err := g.RecordPathByHost(hostname)
	if err != nil {
		return nil, err
	}
	str, err := g.etcdClient.Get(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(str, ";"), nil
}

// 以下三个函数使用闭包的方式实现单例模式，保证在整个程序运行期间只有一个 Set、Get 和 Release 实例
var getSet = func() func() *Set {
	var _set *Set
	return func() *Set {
		if _set != nil {
			return _set
		}
		_set = &Set{}
		_set.etcdClient = getEtcdClient()
		_set.k8sClient = getLightK8sClient()
		return _set
	}
}()

var getGet = func() func() *Get {
	var _get *Get
	return func() *Get {
		if _get != nil {
			return _get
		}
		_get = &Get{
			cidrCache:   map[string]string{},
			nodeIpCache: map[string]string{},
		}
		_get.etcdClient = getEtcdClient()
		_get.k8sClient = getLightK8sClient()
		return _get
	}
}()

var getRelase = func() func() *Release {
	var _release *Release
	return func() *Release {
		if _release != nil {
			return _release
		}
		_release = &Release{}
		_release.etcdClient = getEtcdClient()
		_release.k8sClient = getLightK8sClient()
		return _release
	}
}()

// isGatewayIP 函数用于检查给定的 IP 是否为网关 IP（每个网段的 x.x.x.1）
func isGatewayIP(ip string) bool {
	// 把每个网段的 x.x.x.1 当做网关
	if ip == "" {
		return false
	}
	_arr := strings.Split(ip, ".")
	return _arr[3] == "1"
}

// isRetainIP 函数用于检查给定的 IP 是否为保留 IP（每个网段的 x.x.x.0）
func isRetainIP(ip string) bool {
	// 把每个网段的 x.x.x.0 当做保留
	if ip == "" {
		return false
	}
	_arr := strings.Split(ip, ".")
	return _arr[3] == "0"
}

// 将参数的 IPs 设置到 etcd 中。首先获取当前主机对应的网段，然后获取当前主机的网段下所有已经使用的 IP。遍历给定的 IPs，如果不存在于已使用的 IP 列表中，将其添加到 etcd。
func (s *Set) IPs(ips ...string) error {
	defer unlock()
	// 先拿到当前主机对应的网段
	currentNetwork, err := s.etcdClient.Get(getHostPath())
	if err != nil {
		return err
	}
	// 拿到当前主机的网段下所有已经使用的 ip
	allUsedIPs, err := s.etcdClient.Get(getRecordPath(currentNetwork))
	if err != nil {
		return err
	}
	_allUsedIPsArr := strings.Split(allUsedIPs, ";")
	_tempIPs := allUsedIPs
	for _, ip := range ips {
		if _tempIPs == "" {
			_tempIPs = ip
		} else {
			flag := true
			for i := 0; i < len(_allUsedIPsArr); i++ {
				if _allUsedIPsArr[i] == ip {
					// 如果 etcd 上已经存了则不用再写入了
					flag = false
					break
				}
			}
			if flag {
				_tempIPs += ";" + ip
			}
		}
	}

	return s.etcdClient.Set(getRecordPath(currentNetwork), _tempIPs)
}

// 根据主机名获取一个当前主机可用的网段。如果主机对应的网段已存在，直接返回该网段；否则，从可用的 IP 池中选取一个网段，并更新 etcd 中的 IP 池。如果提供了 IP 地址范围，创建一个范围目录。
func (is *IpamService) networkInit(hostPath, poolPath string, ranges ...string) (string, error) {
	lock()
	defer unlock()
	network, err := is.EtcdClient.Get(hostPath)
	if err != nil {
		return "", err
	}

	// 已经存过该主机对应的网段了
	if network != "" {
		return network, nil
	}

	// 从可用的 ip 池中捞一个
	pool, err := is.EtcdClient.Get(poolPath)
	if err != nil {
		return "", err
	}

	_tempIPs := strings.Split(pool, ";")
	tmpRandom := utils2.GetRandomNumber(len(_tempIPs))
	// TODO: 这块还是得想办法加锁
	currentHostNetwork := _tempIPs[tmpRandom]
	newTmpIps := append([]string{}, _tempIPs[0:tmpRandom]...)
	_tempIPs = append(newTmpIps, _tempIPs[tmpRandom+1:]...)
	// 先把 pool 更新一下
	err = is.EtcdClient.Set(poolPath, strings.Join(_tempIPs, ";"))
	if err != nil {
		return "", err
	}
	// 再把这个网段存到对应的这台主机的 key 下
	err = is.EtcdClient.Set(hostPath, currentHostNetwork)
	if err != nil {
		return "", err
	}

	// 如果传了 ip 地址的 range 的话就创建一个 range 目录
	start := ""
	end := ""
	switch len(ranges) {
	case 1:
		start = ranges[0]
	case 2:
		start = ranges[0]
		end = ranges[1]
	}

	if start != "" && end != "" {
		ranges := utils2.GenIpRange(start, end)
		if ranges != nil {
			currentIpRanges := strings.Join(utils2.GenIpRange(start, end), ";")
			err = is.EtcdClient.Set(fmt.Sprintf(
				"%s/%s/range",
				hostPath,
				currentHostNetwork,
			), currentIpRanges)
			if err != nil {
				return "", err
			}
		}
	}

	return currentHostNetwork, nil
}

// 获取主机名和网段的映射。从 etcd 获取映射信息，将其反序列化为 Go map，并返回。
func (is *IpamService) getHostSubnetMap() (map[string]string, error) {
	path, err := is.Get().HostSubnetMapPath()
	if err != nil {
		return nil, err
	}

	_maps, err := is.EtcdClient.Get(path)
	if err != nil {
		return nil, err
	}

	resMaps := map[string]string{}
	err = json.Unmarshal(([]byte)(_maps), &resMaps)
	if err != nil {
		return nil, err
	}
	return resMaps, nil
}

// 初始化主机名和网段的映射。如果映射中已存在当前子网，直接返回；否则，将当前子网与主机名的映射添加到 etcd 中。
func (is *IpamService) subnetMapInit(subnet, mask, hostname, currentSubnet string) error {
	lock()
	defer unlock()
	m := fmt.Sprintf("/%s/%s/maps", subnet, mask)
	path := getEtcdPathWithPrefix(m)
	maps, err := is.EtcdClient.Get(path)
	if err != nil {
		return err
	}

	if len(maps) == 0 {
		_maps := map[string]string{}
		_maps[currentSubnet] = hostname
		mapsStr, err := json.Marshal(_maps)
		if err != nil {
			return err
		}
		return is.EtcdClient.Set(path, string(mapsStr))
	}

	_tmpMaps := map[string]string{}
	err = json.Unmarshal(([]byte)(maps), &_tmpMaps)
	if err != nil {
		return err
	}

	if _, ok := _tmpMaps[currentSubnet]; ok {
		return nil
	}
	_tmpMaps[currentSubnet] = hostname
	mapsStr, err := json.Marshal(_tmpMaps)
	if err != nil {
		return err
	}
	return is.EtcdClient.Set(path, string(mapsStr))
}

/**
 * 初始化 IP 网段池。如果网段池已存在，直接返回；否则，创建 255 个备用网段，并将其存储到 etcd 中。
 * 比如 subnet 是 10.244.0.0, mask 是 24 的话
 * 就会在 etcd 中初始化出一个
 * 	10.244.0.0;10.244.1.0;10.244.2.0;......;10.244.254.0;10.244.255.0
 */
func (is *IpamService) ipsPoolInit(poolPath string) error {
	lock()
	defer unlock()
	val, err := is.EtcdClient.Get(poolPath)
	if err != nil {
		return err
	}
	if len(val) > 0 {
		return nil
	}
	subnet := is.Subnet
	_temp := strings.Split(subnet, ".")
	_tempIndex := 0
	for _i := 0; _i < len(_temp); _i++ {
		if _temp[_i] == "0" {
			// 找到 subnet 中第一个 0 的位置
			_tempIndex = _i
			break
		}
	}
	/**
	 * FIXME: 对于子网网段的创建, 其实可以不完全是 8 的倍数
	 * 比如 10.244.0.0/26 这种其实也可以
	 */
	// 创建出 255 个备用的网段
	// 每个节点从这些网段中选择一个还没有使用过的
	_tempIpStr := ""
	for _j := 0; _j <= 255; _j++ {
		_temp[_tempIndex] = fmt.Sprintf("%d", _j)
		_newIP := strings.Join(_temp, ".")
		if _tempIpStr == "" {
			_tempIpStr = _newIP
		} else {
			_tempIpStr += ";" + _newIP
		}
	}
	return is.EtcdClient.Set(poolPath, _tempIpStr)
}

/**
 * 获取集群中全部的主机名。从 etcd 的 key 下获取全部节点的 key，而不是调用 Kubernetes API。
 * 这里直接从 etcd 的 key 下边查
 */
func (g *Get) NodeNames() ([]string, error) {
	defer unlock()
	const _minionsNodePrefix = "/registry/minions/"

	nodes, err := g.etcdClient.GetAllKey(_minionsNodePrefix, oriEtcd.WithKeysOnly(), oriEtcd.WithPrefix())

	if err != nil {
		utils2.WriteLog("这里从 etcd 获取全部 nodes key 失败, err: ", err.Error())
		return nil, err
	}

	var res []string
	for _, node := range nodes {
		node = strings.Replace(node, _minionsNodePrefix, "", 1)
		res = append(res, node)
	}
	return res, nil
}

/**
 * 获取集群中全部节点的网络信息。遍历所有节点，获取每个节点的 IP 地址、CIDR 信息等，并将其封装为一个 Network 结构体的切片
 */
func (g *Get) AllHostNetwork() ([]*Network, error) {
	names, err := g.NodeNames()
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	res := []*Network{}
	for _, name := range names {
		ip, err := g.NodeIp(name)
		if err != nil {
			return nil, err
		}

		cidr, err := g.CIDR(name)
		if err != nil {
			return nil, err
		}

		if name == hostname {
			res = append(res, &Network{
				Hostname:      name,
				IP:            ip,
				IsCurrentHost: true,
				CIDR:          cidr,
			})
		} else {
			res = append(res, &Network{
				Hostname:      name,
				IP:            ip,
				IsCurrentHost: false,
				CIDR:          cidr,
			})
		}
	}
	return res, nil
}

/**
 * 获取集群中除了本机以外的全部节点的网络信息。这个函数首先获取集群中所有节点的网络信息，然后过滤掉本机的网络信息，最后将其他节点的网络信息存储在一个数组中并返回。
 */
func (g *Get) AllOtherHostNetwork() ([]*Network, error) {
	networks, err := g.AllHostNetwork()
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	result := []*Network{}
	for _, network := range networks {
		if network.Hostname == hostname {
			continue
		}
		result = append(result, network)
	}
	return result, nil
}

/*
*
  - 获取集群中除了本机以外的全部节点的 IP。这个函数首先获取本机的主机名，然后通过 Kubernetes API
    获取集群中所有节点的信息。接着，遍历每个节点的地址信息，找到内部 IP 地址，并将其存储在一个映射中，最后返回该映射。
*/
func (g *Get) AllOtherHostIP() (map[string]string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	nodes, err := g.k8sClient.Get().Nodes()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(nodes.Items)-1)
	for _, node := range nodes.Items {
		ip := ""
		_hostname := ""
	iternal:
		for _, addr := range node.Status.Addresses {
			if addr.Type == "Hostname" {
				if addr.Address == hostname {
					ip = ""
					_hostname = ""
					break iternal
				}
				_hostname = addr.Address
			}
			if addr.Type == "InternalIP" {
				ip = addr.Address
			}
		}
		if ip != "" {
			result[_hostname] = ip
		}
	}
	return result, nil
}

/**
* 获取本机网卡的信息。这个函数首先获取本机的所有网络相关设备，然后获取本机的主机名。
* 接着，使用主机名从 IPAM 服务中获取本机的 IP 地址。遍历所有网络设备，找到类型为 "device" 的设备，
* 并获取其 IPv4 地址。如果找到与主机 IP 相匹配的设备，将其相关信息存储在一个 Network 结构中并返回。
 */
func (g *Get) HostNetwork() (*Network, error) {
	// 先拿到本机上所有的网络相关设备
	linkList, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	// 先获取一下 ipam
	ipam, err := GetIpamService()
	if err != nil {
		return nil, err
	}
	// 然后拿本机的 hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// 用这个 hostname 获取本机的 ip
	hostIP, err := ipam.Get().NodeIp(hostname)
	if err != nil {
		return nil, err
	}
	for _, link := range linkList {
		// 就看类型是 device 的
		if link.Type() == "device" {
			// 找每块儿设备的地址信息
			addr, err := netlink.AddrList(link, netlink.FAMILY_V4)
			if err != nil {
				continue
			}
			if len(addr) >= 1 {
				// TODO: 这里其实应该处理一下一块儿网卡绑定了多个 ip 的情况
				// 数组中的每项都是这样的格式 "192.168.98.143/24 ens33"
				_addr := strings.Split(addr[0].String(), " ")
				ip := _addr[0]
				name := _addr[1]
				ip = strings.Split(ip, "/")[0]
				if ip == hostIP {
					// 走到这儿说明主机走的就是这块儿网卡
					return &Network{
						Name:          name,
						IP:            hostIP,
						Hostname:      hostname,
						IsCurrentHost: true,
					}, nil
				}
			}
		}
	}
	return nil, errors.New("no valid network device found")
}

// 根据主机名获取当前节点被分配到的网段和掩码。这个函数首先从缓存中查找 CIDR，如果找到则直接返回。
// 如果缓存中没有，则从 Etcd 中获取 CIDR 信息，并将其与 IPAM 服务中的 PodMaskSegment
// 拼接成完整的 CIDR。将结果存储在缓存中并返回。
func (g *Get) CIDR(hostName string) (string, error) {
	defer unlock()
	if val, ok := g.cidrCache[hostName]; ok {
		return val, nil
	}
	_cidrPath := getEtcdPathWithPrefix("/" + getIpamSubnet() + "/" + getIpamMaskSegment() + "/" + hostName)

	etcd := getEtcdClient()
	if etcd == nil {
		return "", errors.New("etcd client not found")
	}

	cidr, err := etcd.Get(_cidrPath)
	if err != nil {
		return "", err
	}

	if cidr == "" {
		return "", nil
	}

	// 先获取一下 ipam
	ipam, err := GetIpamService()
	if err != nil {
		return "", err
	}
	cidr += ("/" + ipam.PodMaskSegment)
	g.cidrCache[hostName] = cidr
	return cidr, nil
}

/*
* 根据主机名获取节点 IP。这个函数首先从缓存中查找节点 IP，如果找到则直接返回。
如果缓存中没有，则通过 Kubernetes API 获取节点信息，并遍历节点的地址信息以找到内部 IP 地址。
将结果存储在缓存中并返回。
*/
func (g *Get) NodeIp(hostName string) (string, error) {
	defer unlock()
	if val, ok := g.nodeIpCache[hostName]; ok {
		return val, nil
	}
	node, err := g.k8sClient.Get().Node(hostName)
	if err != nil {
		return "", err
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == "InternalIP" {
			g.nodeIpCache[hostName] = addr.Address
			return addr.Address, nil
		}
	}
	return "", errors.New("没有找到 ip")
}

// 获取下一个未使用的 IP 地址。这个函数首先获取当前网络的信息和所有已使用的 IP 地址。
// 然后，尝试从 IP 范围中随机选取一个未使用的 IP。如果 IP 范围不存在或无法访问，
// 该函数将从默认网关附近的 IP 地址中随机选择一个未使用的 IP。
func (g *Get) nextUnusedIP() (string, error) {
	defer unlock()
	currentNetwork, err := g.etcdClient.Get(getHostPath())
	if err != nil {
		return "", err
	}
	allUsedIPs, err := g.etcdClient.Get(getRecordPath(currentNetwork))
	if err != nil {
		return "", err
	}

	ipsMap := map[string]bool{}
	ips := strings.Split(allUsedIPs, ";")
	for _, ip := range ips {
		ipsMap[ip] = true
	}

	if rangesPathExist, err := g.etcdClient.GetKey(getIpRangesPath(currentNetwork)); rangesPathExist != "" && err == nil {
		if rangesIPs, err := g.etcdClient.Get(getIpRangesPath(currentNetwork)); err == nil {
			rangeIpsArr := strings.Split(rangesIPs, ";")
			if len(rangeIpsArr) == 0 {
				return "", errors.New("all of the ips are used")
			}
			nextIp := ""
			for {
				if len(ipsMap) == len(rangeIpsArr) {
					return "", errors.New("all of the ips are used")
				}
				nextIp = ""
				for i, ip := range rangeIpsArr {
					if utils2.GetRandomNumber(i+1) == 0 {
						nextIp = ip
					}
				}
				if _, ok := ipsMap[nextIp]; !ok {
					break
				}
			}

			return nextIp, nil
		}
	}

	gw, err := g.Gateway()
	if err != nil {
		return "", err
	}
	nextIp := ""
	gwNum := utils2.InetIP2Int(gw)
	for {
		n := utils2.GetRandomNumber(254)
		if n == 0 || n == 1 {
			continue
		}
		nextIpNum := gwNum + int64(n)
		nextIp = utils2.InetInt2Ip(nextIpNum)
		if _, ok := ipsMap[nextIp]; !ok {
			break
		}
	}

	return nextIp, nil
}

// 获取当前网络的网关 IP。这个函数首先从 Etcd 中获取当前网络的信息，
// 然后将当前网络的 IP 地址加 1 作为网关 IP，并将其转换为字符串格式返回。
func (g *Get) Gateway() (string, error) {
	defer unlock()
	currentNetwork, err := g.etcdClient.Get(getHostPath())
	if err != nil {
		return "", err
	}

	return utils2.InetInt2Ip((utils2.InetIP2Int(currentNetwork) + 1)), nil
}

// 获取当前网络的网关 IP 以及掩码段。这个函数首先获取当前网络的网关 IP，
// 然后将其与 IPAM 服务中的掩码段拼接起来，形成一个完整的网关 IP 和掩码段字符串并返回。
func (g *Get) GatewayWithMaskSegment() (string, error) {
	defer unlock()
	currentNetwork, err := g.etcdClient.Get(getHostPath())
	if err != nil {
		return "", err
	}

	return utils2.InetInt2Ip((utils2.InetIP2Int(currentNetwork) + 1)) + "/" + getIpamMaskSegment(), nil
}

// 获取所有已使用的 IP 地址。这个函数首先从 Etcd 中获取当前网络的信息和所有已使用的 IP 地址，
// 然后将所有已使用的 IP 地址以字符串数组的形式返回。
func (g *Get) AllUsedIPs() ([]string, error) {
	defer unlock()
	currentNetwork, err := g.etcdClient.Get(getHostPath())
	if err != nil {
		return nil, err
	}
	allUsedIPs, err := g.etcdClient.Get(getRecordPath(currentNetwork))
	if err != nil {
		return nil, err
	}
	return strings.Split(allUsedIPs, ";"), nil
}

// 根据主机名获取该主机上所有已使用的 IP 地址。
// 这个函数首先从 Etcd 中获取当前网络的信息和所有已使用的 IP 地址，
// 然后将所有已使用的 IP 地址以字符串数组的形式返回。
func (g *Get) AllUsedIPsByHost(hostname string) ([]string, error) {
	defer unlock()
	currentNetwork, err := g.etcdClient.Get(getHostPath())
	if err != nil {
		return nil, err
	}
	allUsedIPs, err := g.etcdClient.Get(getRecordPath(currentNetwork))
	if err != nil {
		return nil, err
	}
	return strings.Split(allUsedIPs, ";"), nil
}

// 获取一个未使用的 IP 地址。这个函数会循环尝试获取下一个未使用的 IP 地址，
// 直到找到一个有效的 IP。如果找到的 IP 是网关 IP 或保留 IP，该函数将尝试将其标记为已使用，
// 然后继续查找下一个未使用的 IP。一旦找到一个有效的未使用 IP，该函数将其标记为已使用，并将其返回。
func (g *Get) UnusedIP() (string, error) {
	defer unlock()
	for {
		ip, err := g.nextUnusedIP()
		if err != nil {
			return "", err
		}
		if isGatewayIP(ip) || isRetainIP(ip) {
			err = getSet().IPs(ip)
			if err != nil {
				return "", err
			}
			continue
		}
		// 先把这个 ip 占上坑位
		// 坑位先占上不影响大局
		// 但是如果坑位占晚了被别人抢先的话可能会导致有俩 pod 的 ip 冲突
		err = getSet().IPs(ip)
		if err != nil {
			return "", err
		}
		return ip, nil
	}
}

/*
*
  - 这个函数用于释放一组 IP 地址。它首先从 Etcd 中获取当前主机的网络信息和已使用的 IP 地址。

然后，将要释放的 IP 地址从已使用的 IP 地址中移除，并将结果重新写入 Etcd。
*/
func (r *Release) IPs(ips ...string) error {
	defer unlock()
	currentNetwork, err := r.etcdClient.Get(getHostPath())
	if err != nil {
		return err
	}
	allUsedIPs, err := r.etcdClient.Get(getRecordPath(currentNetwork))
	if err != nil {
		return err
	}
	_allUsedIP := strings.Split(allUsedIPs, ";")
	var _newIPs []string
	for _, usedIP := range _allUsedIP {
		flag := false
		for _, ip := range ips {
			if usedIP == ip {
				flag = true
				break
			}
		}
		if !flag {
			_newIPs = append(_newIPs, usedIP)
		}
	}
	newIPsString := strings.Join(_newIPs, ";")
	return r.etcdClient.Set(getRecordPath(currentNetwork), newIPsString)
}

// 这个函数用于释放 IP 池。它首先从 Etcd 中获取当前 IP 池的网络信息，然后将其设置为空字符串。
func (r *Release) Pool() error {
	defer unlock()
	currentNetwork, err := r.etcdClient.Get(getIPsPoolPath(getIpamSubnet(), getIpamMaskSegment()))
	if err != nil {
		return err
	}

	return r.etcdClient.Set(currentNetwork, "")
}

// 这个函数用于获取 IPAM 服务的 Get 实例。它首先加锁，然后返回 Get 实例。
func (o *operator) Get() *Get {
	lock()
	return getGet()
}

// 这个函数用于获取 IPAM 服务的 Set 实例。它首先加锁，然后返回 Set 实例。
func (o *operator) Set() *Set {
	lock()
	return getSet()
}

// 这个函数用于获取 IPAM 服务的 Release 实例。它首先加锁，然后返回 Release 实例。
func (o *operator) Release() *Release {
	lock()
	return getRelase()
}

// 这个函数用于根据给定的路径生成一个带有前缀的 Etcd 路径。
func getEtcdPathWithPrefix(path string) string {
	if path != "" && path[0:1] == "/" {
		return "/" + prefix + path
	}
	return "/" + prefix + "/" + path
}

// 这个函数根据掩码的位数返回相应的子网掩码 IP 地址。
func getMaskIpFromNum(numStr string) string {
	switch numStr {
	case "8":
		return "255.0.0.0"
	case "16":
		return "255.255.0.0"
	case "24":
		return "255.255.255.0"
	case "32":
		return "255.255.255.255"
	default:
		return "255.255.0.0"
	}
}

// 这个函数是一个闭包，用于初始化 IPAM 服务。它返回一个函数，该函数创建并返回一个
// IpamService 实例。在创建过程中，它会处理子网参数、掩码、网段范围
// 等。此外，它还会初始化 Etcd 客户端、K8s 客户端、IP 池以及主机可用的网络等。
var __GetIpamService func() (*IpamService, error)

func _GetIpamService(subnet string, options *IPAMOptions) func() (*IpamService, error) {

	return func() (*IpamService, error) {
		var _ipam *IpamService

		if _ipam != nil {
			return _ipam, nil
		} else {
			_subnet := subnet
			var _maskSegment string = consts.DEFAULT_MASK_NUM
			var _podIpMaskSegment string = consts.DEFAULT_MASK_NUM
			var _rangeStart string = ""
			var _rangeEnd string = ""
			if options != nil {
				if options.MaskSegment != "" {
					_maskSegment = options.MaskSegment
				}
				if options.PodIpMaskSegment != "" {
					_podIpMaskSegment = options.PodIpMaskSegment
				}
				if options.RangeStart != "" {
					_rangeStart = options.RangeStart
				}
				if options.RangeEnd != "" {
					_rangeEnd = options.RangeEnd
				}
			}

			// 配置文件中传参数的时候可能直接传了个子网掩码
			// 传了的话就直接使用这个掩码
			if withMask := strings.Contains(subnet, "/"); withMask {
				subnetAndMask := strings.Split(subnet, "/")
				_subnet = subnetAndMask[0]
				_maskSegment = subnetAndMask[1]
			}

			var _maskIP string = getMaskIpFromNum(_maskSegment)
			var _podMaskIP string = getMaskIpFromNum(_podIpMaskSegment)

			// 如果不是合法的子网地址的话就强转成合法
			// 比如 _subnet 传了个数字过来, 要给它先干成 a.b.c.d 的样子
			// 然后 & maskIP, 给做成类似 a.b.0.0 的样子
			_subnet = utils2.InetInt2Ip(utils2.InetIP2Int(_subnet) & utils2.InetIP2Int(_maskIP))
			_ipam = &IpamService{
				Subnet:         _subnet,           // 子网网段
				MaskSegment:    _maskSegment,      // 掩码 10 进制
				MaskIP:         _maskIP,           // 掩码 ip
				PodMaskSegment: _podIpMaskSegment, // pod 的 mask 10 进制
				PodMaskIP:      _podMaskIP,        // pod 的 mask ip
			}
			_ipam.EtcdClient = getEtcdClient()
			_ipam.K8sClient = getLightK8sClient()
			// 初始化一个 ip 网段的 pool
			// 如果已经初始化过就不再初始化
			poolPath := getEtcdPathWithPrefix("/" + _ipam.Subnet + "/" + _ipam.MaskSegment + "/" + "pool")
			err := _ipam.ipsPoolInit(poolPath)
			if err != nil {
				return nil, err
			}

			// 然后尝试去拿一个当前主机可用的网段
			// 如果拿不到, 里面会尝试创建一个
			hostname, err := os.Hostname()
			if err != nil {
				return nil, err
			}
			hostPath := getEtcdPathWithPrefix("/" + _ipam.Subnet + "/" + _ipam.MaskSegment + "/" + hostname)
			currentHostNetwork, err := _ipam.networkInit(
				hostPath,
				poolPath,
				_rangeStart,
				_rangeEnd,
			)
			if err != nil {
				return nil, err
			}

			// 初始化一个 map 的地址给 ebpf 用
			err = _ipam.subnetMapInit(
				_subnet,
				_maskSegment,
				hostname,
				currentHostNetwork,
			)
			if err != nil {
				return nil, err
			}

			_ipam.CurrentHostNetwork = currentHostNetwork
			return _ipam, nil
		}
	}
}

// 这个函数用于获取 IPAM 服务的实例。如果服务未初始化，将返回一个错误。
func GetIpamService() (*IpamService, error) {
	if __GetIpamService == nil {
		return nil, errors.New("ipam service 需要初始化")
	}

	ipamService, err := __GetIpamService()
	if err != nil {
		return nil, err
	}
	return ipamService, nil
}

// 这个函数用于清除 IPAM 服务的实例，并从 Etcd 中删除所有与之相关的键。
func (is *IpamService) clear() error {
	__GetIpamService = nil
	return is.EtcdClient.Del("/"+prefix, oriEtcd.WithPrefix())
}

// 这个函数用于初始化 IPAM 服务。它首先检查服务是否已经初始化，如果没有，则调用 _GetIpamService() 函数进行初始化。
// 然后，返回一个函数，该函数用于清除 IPAM 服务的实例并从 Etcd 中删除所有与之相关的键。
func Init(subnet string, options *IPAMOptions) func() error {
	if __GetIpamService == nil {
		__GetIpamService = _GetIpamService(subnet, options)
	}
	is, err := GetIpamService()
	if err != nil {
		return func() error {
			return err
		}
	}
	return is.clear
}
