package client

import (
	"cni-demo/consts"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"net/http"
)

type Get struct {
	httpsClient *http.Client    // https 客户端
	client      *LightK8sClient // 自定义 K8s 客户端
}

type operators struct {
	Get *Get // Get 操作
}

type operator struct {
	*operators // operators 结构体指针
}

// LightK8sClient 是一个自定义的 Kubernetes 客户端，用于与 K8s API Server 进行交互
type LightK8sClient struct {
	caCertPath, certFile, keyFile string         // 证书文件路径
	pool                          *x509.CertPool // 证书池
	client                        *http.Client   // http 客户端
	masterEndpoint                string         // Kubernetes API Server 地址
	kubeApi                       string         // Kubernetes API
	*operator                                    // operator 结构体指针
}

// getGet 函数返回一个 Get 类型的单例
var getGet = func() func() *Get {
	var _get *Get
	return func() *Get {
		if _get != nil {
			return _get
		}
		_get = &Get{}
		client, _ := GetLightK8sClient()
		if client != nil {
			_get.httpsClient = client.client
		}
		_get.client = client
		return _get
	}
}()

// getRoute 函数用于生成 API 请求的 URL
func (get *Get) getRoute(api string) string {
	return get.client.masterEndpoint + get.client.kubeApi + api
}

// Get 方法返回一个 Get 结构体
func (o *operator) Get() *Get {
	return getGet()
}

// getBody 函数用于从 http.Response 中读取响应体并返回
func (get *Get) getBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Nodes 方法用于获取集群中所有节点的信息
func (get *Get) Nodes() (*v1.NodeList, error) {
	url := get.getRoute("/nodes?limit=500")
	resp, err := get.httpsClient.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := get.getBody(resp)
	if err != nil {
		return nil, err
	}
	var nodes *v1.NodeList
	err = json.Unmarshal(body, &nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// Node 方法用于获取指定节点的信息
func (get *Get) Node(name string) (*v1.Node, error) {
	url := get.getRoute(fmt.Sprintf("/nodes/%s", name))
	resp, err := get.httpsClient.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := get.getBody(resp)
	if err != nil {
		return nil, err
	}
	var node *v1.Node
	err = json.Unmarshal(body, &node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

var __GetLightK8sClient func() (*LightK8sClient, error)

// _GetLightK8sClient 函数用于初始化 LightK8sClient
func _GetLightK8sClient(caCertPath, certFile, keyFile string) func() (*LightK8sClient, error) {
	return func() (*LightK8sClient, error) {
		var client *LightK8sClient
		if client != nil {
			return client, nil
		} else {
			// 读取 k8s 的证书
			caCrt, err := ioutil.ReadFile(caCertPath)
			if err != nil {
				return nil, err
			}
			// new 个 pool
			pool := x509.NewCertPool()
			// 解析一系列PEM编码的证书, 从 base64 中解析证书到池子中
			pool.AppendCertsFromPEM(caCrt)
			// 加载客户端的证书和私钥
			cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return nil, err
			}
			// 创建一个 https 客户端
			_client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:      pool,
						Certificates: []tls.Certificate{cliCrt},
					},
				},
			}

			masterEndpoint, err := GetMasterEndpoint()
			if err != nil {
				return nil, err
			}
			client = &LightK8sClient{
				caCertPath:     caCertPath,
				certFile:       certFile,
				keyFile:        keyFile,
				pool:           pool,
				client:         _client,
				kubeApi:        consts.KUBE_API,
				masterEndpoint: masterEndpoint,
			}

			return client, nil
		}
	}
}

// GetLightK8sClient 函数用于获取 LightK8sClient 实例
func GetLightK8sClient() (*LightK8sClient, error) {
	if __GetLightK8sClient == nil {
		return nil, errors.New("k8s clinet 需要初始化")
	}

	lightK8sClient, err := __GetLightK8sClient()
	if err != nil {
		return nil, err
	}
	return lightK8sClient, nil
}

// Init 函数用于初始化 LightK8sClient
func Init(caCertPath, certFile, keyFile string) {
	if __GetLightK8sClient == nil {
		__GetLightK8sClient = _GetLightK8sClient(caCertPath, certFile, keyFile)
	}
}
