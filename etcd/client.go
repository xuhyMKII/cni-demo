package etcd

import (
	"cni-demo/client"
	"cni-demo/tools/utils"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	etcd "go.etcd.io/etcd/client/v3"
)

// WatchCallback 是一个回调函数类型，用于处理 etcd watch 事件。
type WatchCallback func(_type mvccpb.Event_EventType, key, value []byte)

// EtcdConfig 结构体用于定义 etcd 客户端的相关配置。
type EtcdConfig struct {
	EtcdScheme       string `json:"etcdScheme" envconfig:"APIV1_ETCD_SCHEME" default:""`
	EtcdAuthority    string `json:"etcdAuthority" envconfig:"APIV1_ETCD_AUTHORITY" default:""`
	EtcdEndpoints    string `json:"etcdEndpoints" envconfig:"APIV1_ETCD_ENDPOINTS"`
	EtcdDiscoverySrv string `json:"etcdDiscoverySrv" envconfig:"APIV1_ETCD_DISCOVERY_SRV"`
	EtcdUsername     string `json:"etcdUsername" envconfig:"APIV1_ETCD_USERNAME"`
	EtcdPassword     string `json:"etcdPassword" envconfig:"APIV1_ETCD_PASSWORD"`
	EtcdKeyFile      string `json:"etcdKeyFile" envconfig:"APIV1_ETCD_KEY_FILE"`
	EtcdCertFile     string `json:"etcdCertFile" envconfig:"APIV1_ETCD_CERT_FILE"`
	EtcdCACertFile   string `json:"etcdCACertFile" envconfig:"APIV1_ETCD_CA_CERT_FILE"`
}

// EtcdConfig 结构体用于定义 etcd 客户端的相关配置。
type Watcher struct {
	client        *EtcdClient
	watcher       etcd.Watcher
	cancelWatcher context.CancelFunc
	ctx           context.Context
	// kMap map[string]
}

// EtcdClient 结构体用于封装 etcd 客户端及其操作。
type EtcdClient struct {
	client  *etcd.Client
	watcher *Watcher
	Version string
}

const (
	clientTimeout = 30 * time.Second
	etcdTimeout   = 2 * time.Second
)

// newEtcdClient 根据给定的配置创建一个新的 etcd 客户端。
func newEtcdClient(config *EtcdConfig) (*etcd.Client, error) {
	var etcdLocation []string
	if config.EtcdAuthority != "" {
		etcdLocation = []string{config.EtcdScheme + "://" + config.EtcdAuthority}
	}
	if config.EtcdEndpoints != "" {
		etcdLocation = strings.Split(config.EtcdEndpoints, ",")
	}

	if len(etcdLocation) == 0 {
		return nil, errors.New("找不到 etcd")
	}

	tlsInfo := transport.TLSInfo{
		CertFile:      config.EtcdCertFile,
		KeyFile:       config.EtcdKeyFile,
		TrustedCAFile: config.EtcdCACertFile,
	}

	tlsConfig, err := tlsInfo.ClientConfig()

	client, err := etcd.New(etcd.Config{
		Endpoints:   etcdLocation,
		TLS:         tlsConfig,
		DialTimeout: clientTimeout,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

var __GetEtcdClient func() (*EtcdClient, error)

// GetEtcdClient 函数用于获取 etcd 客户端实例。
func GetEtcdClient() (*EtcdClient, error) {

	if __GetEtcdClient == nil {
		return nil, nil
	}
	return __GetEtcdClient()

}

// _GetEtcdClient 函数用于初始化并返回一个 etcd 客户端实例。
func _GetEtcdClient() func() (*EtcdClient, error) {
	var _client *EtcdClient

	return func() (*EtcdClient, error) {
		if _client != nil {
			return _client, nil
		} else {
			// curl https://192.168.64.19:6443/openapi/v2 --cacert ./ca.crt --cert ./tmp_ca.crt --key ./tmp.key
			// ETCDCTL_API=3 etcdctl --endpoints https://192.168.64.19:2379 --cacert /etc/kubernetes/pki/etcd/ca.crt --cert /etc/kubernetes/pki/etcd/healthcheck-client.crt --key /etc/kubernetes/pki/etcd/healthcheck-client.key get / --prefix --keys-only

			configPath := client.GetClientConfigPath()
			confByte, err := ioutil.ReadFile(configPath)
			if err != nil {
				panic(fmt.Sprintf("读取 path: %s 失败: %s", configPath, err.Error()))
			}
			master, err := client.GetLineFromYaml(string(confByte), "server")
			if err != nil {
				panic(fmt.Sprintf("在 etcd 初始化时尝试获取 master 节点失败: %s", err.Error()))
			}
			etcdEp := ""
			if master != "" {
				masteIp := strings.Split(master, ":")
				if len(masteIp) == 3 {
					etcdEp = fmt.Sprintf("%s:%s:2379", masteIp[0], masteIp[1])
				}
			}
			if os.Getenv("ETCD_ENDPOINT") != "" {
				etcdEp = os.Getenv("ETCD_ENDPOINT")
			}
			if etcdEp == "" {
				panic("get etcd endpoint failed from env")
			}
			client, err := newEtcdClient(&EtcdConfig{
				EtcdEndpoints:  etcdEp,
				EtcdCertFile:   "/etc/kubernetes/pki/etcd/healthcheck-client.crt",
				EtcdKeyFile:    "/etc/kubernetes/pki/etcd/healthcheck-client.key",
				EtcdCACertFile: "/etc/kubernetes/pki/etcd/ca.crt",
			})

			if err != nil {
				return nil, err
			}

			status, err := client.Status(context.TODO(), etcdEp)

			if err != nil {
				utils.WriteLog("无法获取到 etcd 版本")
				return nil, err
			}

			if client != nil {
				_client = &EtcdClient{
					client: client,
				}

				if status != nil && status.Version != "" {
					_client.Version = status.Version
				}
				// fmt.Println("etcd 客户端初始化成功")
				return _client, nil
			}
		}
		return nil, errors.New("初始化 etcd client 失败")
	}
}

// Init 函数用于初始化 etcd 客户端。
func Init() {
	if __GetEtcdClient == nil {
		__GetEtcdClient = _GetEtcdClient()
	}
}

// Set 方法用于在 etcd 中设置一个键值对。
func (c *EtcdClient) Set(key, value string) error {
	_, err := c.client.Put(context.TODO(), key, value)

	if err != nil {
		return err
	}
	return nil
}

// Del 方法用于在 etcd 中删除一个键。
func (c *EtcdClient) Del(key string, opts ...etcd.OpOption) error {
	_, err := c.client.Delete(context.TODO(), key, opts...)
	if err != nil {
		return err
	}
	return err
}

// GetVersion 方法用于获取 etcd 中某个键的版本信息。
func (c *EtcdClient) GetVersion(key string, opts ...etcd.OpOption) (int64, error) {
	resp, err := c.client.Get(context.TODO(), key, opts...)
	if err != nil {
		return 0, err
	}
	if len(resp.Kvs) > 0 {
		return resp.Kvs[len(resp.Kvs)-1:][0].Version, nil
	}
	return 0, nil
}

// Get 方法用于从 etcd 中获取一个键对应的值。
func (c *EtcdClient) Get(key string, opts ...etcd.OpOption) (string, error) {
	resp, err := c.client.Get(context.TODO(), key, opts...)
	if err != nil {
		return "", err
	}

	// for _, ev := range resp.Kvs {
	// 	fmt.Println("这里的 ev 是: ", ev)
	// 	fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	// }

	if len(resp.Kvs) > 0 {
		return string(resp.Kvs[len(resp.Kvs)-1:][0].Value), nil
	}
	return "", nil
}

// GetKey 方法用于从 etcd 中获取一个键。
func (c *EtcdClient) GetKey(key string, opts ...etcd.OpOption) (string, error) {
	resp, err := c.client.Get(context.TODO(), key, opts...)
	if err != nil {
		return "", err
	}

	// for _, ev := range resp.Kvs {
	// 	fmt.Println("这里的 ev 是: ", ev)
	// 	fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	// }

	if len(resp.Kvs) > 0 {
		return string(resp.Kvs[len(resp.Kvs)-1:][0].Key), nil
	}
	return "", nil
}

// GetAll 方法用于从 etcd 中获取所有与给定键匹配的值。
func (c *EtcdClient) GetAll(key string, opts ...etcd.OpOption) ([]string, error) {
	resp, err := c.client.Get(context.TODO(), key, opts...)
	if err != nil {
		return nil, err
	}

	var res []string

	for _, ev := range resp.Kvs {
		// fmt.Println("这里的 ev 是: ", ev)
		// fmt.Printf("%s : %s\n", ev.Key, ev.Value)
		res = append(res, string(ev.Value))
	}

	return res, nil
}

// GetAllKey 方法用于从 etcd 中获取所有与给定键匹配的键。
func (c *EtcdClient) GetAllKey(key string, opts ...etcd.OpOption) ([]string, error) {
	resp, err := c.client.Get(context.TODO(), key, opts...)
	if err != nil {
		return nil, err
	}

	var res []string

	for _, ev := range resp.Kvs {
		res = append(res, string(ev.Key))
	}

	return res, nil
}

// Watch 方法用于在 etcd 中监听一个键的变化，并通过回调函数处理变化事件。
func (c *EtcdClient) Watch(key string, cb WatchCallback) {
	go func() {
		for {
			change := c.client.Watch(context.Background(), key)
			for wresp := range change {
				for _, ev := range wresp.Events {
					cb(ev.Type, ev.Kv.Key, ev.Kv.Value)
				}
			}
		}
	}()
}

// Done 方法返回一个 channel，当 Watcher 结束时会关闭该 channel。
func (w *Watcher) Done() <-chan struct{} {
	return w.ctx.Done()
}

// Deadline 方法返回 Watcher 的截止时间（如果有的话）。
func (w *Watcher) Deadline() (deadline time.Time, ok bool) {
	return w.ctx.Deadline()
}

// Error 方法返回 Watcher 的错误信息（如果有的话）。
func (w *Watcher) Error() error {
	return w.ctx.Err()
}

// Value 方法返回与给定键关联的值（如果有的话）。
func (w *Watcher) Value(_any interface{}) interface{} {
	return w.ctx.Value(_any)
}

// Cancel 方法用于取消 Watcher 的监听操作。
func (w *Watcher) Cancel() {
	w.cancelWatcher()
}

// Watch 方法用于监听一个键的变化，并通过回调函数处理变化事件。
func (w *Watcher) Watch(key string, cb WatchCallback) {
	go func() {
		defer func() {
			w.Cancel()
			time.Sleep(2 * time.Second)
		}()
		for {
			change := w.watcher.Watch(context.Background(), key)
			for wresp := range change {
				for _, ev := range wresp.Events {
					cb(ev.Type, ev.Kv.Key, ev.Kv.Value)
				}
			}
		}
	}()
	// TODO: sleep change to sync
	time.Sleep(1 * time.Second)
}

// GetWatcher 方法返回一个新的 Watcher 实例。
func (c *EtcdClient) GetWatcher() (*Watcher, error) {
	if c.watcher != nil {
		return c.watcher, nil
	}
	watcher := &Watcher{client: c}
	_watcher := etcd.NewWatcher(c.client)
	watcher.watcher = _watcher
	ctx, cancelFunc := context.WithCancel(context.TODO())
	watcher.cancelWatcher = cancelFunc
	watcher.ctx = ctx

	return watcher, nil
}
