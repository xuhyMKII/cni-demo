package client

import (
	"cni-demo/consts"
	utils2 "cni-demo/tools/utils"
	"encoding/base64"
	"fmt"
	"github.com/dlclark/regexp2"
	"io/ioutil"
)

// GetClientConfigPath 函数尝试按以下顺序查找配置文件路径：
// 1. admin.conf
// 2. kubelet.conf
// 3. ~/.kube/conf
// 如果都找不到，则返回默认的 consts.KUBE_LOCAL_DEFAULT_PATH
func GetClientConfigPath() string {
	clusterConfPath := consts.KUBE_LOCAL_DEFAULT_PATH
	if utils2.PathExists(consts.KUBE_CONF_ADMIN_DEFAULT_PATH) {
		clusterConfPath = consts.KUBE_CONF_ADMIN_DEFAULT_PATH
	} else if utils2.PathExists(consts.KUBELET_CONFIG_DEFAULT_PATH) {
		clusterConfPath = consts.KUBELET_CONFIG_DEFAULT_PATH
	}
	return clusterConfPath
}

// GetMasterEndpoint 函数尝试从上述的配置文件路径中读取 master endpoint。
func GetMasterEndpoint() (string, error) {
	// 先尝试去捞 admin.conf, 没有的话就用 kubelet.conf, 再没有的话就用 ~/.kube/conf
	clusterConfPath := GetClientConfigPath()
	confByte, err := ioutil.ReadFile(clusterConfPath)
	if err != nil {
		utils2.WriteLog("读取 path: ", clusterConfPath, " 失败: ", err.Error())
		return "", err
	}
	masterEndpoint, err := GetLineFromYaml(string(confByte), "server")
	if err != nil {
		utils2.WriteLog("读取 path: ", clusterConfPath, " 失败: ", err.Error())
		return "", err
	}
	return masterEndpoint, nil
}

// GetLineFromYaml 函数从给定的 yaml 字符串中根据键值提取对应的内容。
func GetLineFromYaml(yaml string, key string) (string, error) {
	r, err := regexp2.Compile(fmt.Sprintf(`(?<=%s: )(.*)`, key), 0)
	if err != nil {
		utils2.WriteLog("初始化正则表达式失败, err: ", err.Error())
		return "", err
	}

	res, err := r.FindStringMatch(yaml)
	if err != nil {
		utils2.WriteLog("正则匹配 ip 失败, err: ", err.Error())
		return "", err
	}
	return res.String(), nil
}

// AuthenticationInfoPath 结构体用于存储认证信息的路径。
type AuthenticationInfoPath struct {
	CaPath   string // api server 的证书
	CertPath string // 本机的证书
	KeyPath  string // 本机的私钥
}

// GetHostAuthenticationInfoPath 函数获取主机上的认证信息文件的路径。
func GetHostAuthenticationInfoPath() (*AuthenticationInfoPath, error) {
	paths := &AuthenticationInfoPath{}
	if !utils2.PathExists(consts.KUBE_TEST_CNI_DEFAULT_PATH) {
		err := utils2.CreateDir(consts.KUBE_TEST_CNI_DEFAULT_PATH)
		if err != nil {
			return nil, err
		}
	}
	// 如果几个关键的文件已经存在就直接返回路径
	if utils2.PathExists(consts.KUBE_TEST_CNI_TMP_CA_DEFAULT_PATH) {
		paths.CaPath = consts.KUBE_TEST_CNI_TMP_CA_DEFAULT_PATH
	}
	if utils2.PathExists(consts.KUBE_TEST_CNI_TMP_CERT_DEFAULT_PATH) {
		paths.CertPath = consts.KUBE_TEST_CNI_TMP_CERT_DEFAULT_PATH
	}
	if utils2.PathExists(consts.KUBE_TEST_CNI_TMP_KEY_DEFAULT_PATH) {
		paths.KeyPath = consts.KUBE_TEST_CNI_TMP_KEY_DEFAULT_PATH
	}
	if paths.CaPath != "" && paths.CertPath != "" && paths.KeyPath != "" {
		return paths, nil
	}

	var caPath string
	if utils2.PathExists(consts.KUBE_DEFAULT_CA_PATH) {
		caPath = consts.KUBE_DEFAULT_CA_PATH
		err := utils2.FileCopy(caPath, consts.KUBE_TEST_CNI_TMP_CA_DEFAULT_PATH)
		if err != nil {
			return nil, err
		}
		paths.CaPath = consts.KUBE_TEST_CNI_TMP_CA_DEFAULT_PATH
	}
	clusterConfPath := GetClientConfigPath()

	confByte, err := ioutil.ReadFile(clusterConfPath)
	if err != nil {
		utils2.WriteLog("读取 path: ", clusterConfPath, " 失败: ", err.Error())
		return nil, err
	}

	// 根据读取到的配置文件（admin.conf/kubelet.conf/conf中的任何一个），提取客户端证书和key
	// 首先查找是否有 client-certificate-data/client-key-data，如果有则解码并保存到临时目录
	// 如果是 client-certificate/client-key 的话，直接复制到临时目录
	cert, err := GetLineFromYaml(string(confByte), "client-certificate-data")
	if err != nil {
		return nil, err
	}
	key, err := GetLineFromYaml(string(confByte), "client-key-data")
	if err != nil {
		return nil, err
	}
	if cert != "" && key != "" {
		decodedCert, err := base64.StdEncoding.DecodeString(cert)
		if err != nil {
			return nil, err
		}
		err = utils2.CreateFile(consts.KUBE_TEST_CNI_TMP_CERT_DEFAULT_PATH, ([]byte)(decodedCert), 0766)
		if err != nil {
			return nil, err
		}
		decodedKey, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, err
		}
		err = utils2.CreateFile(consts.KUBE_TEST_CNI_TMP_KEY_DEFAULT_PATH, ([]byte)(decodedKey), 0766)
		if err != nil {
			return nil, err
		}
		paths.CertPath = consts.KUBE_TEST_CNI_TMP_CERT_DEFAULT_PATH
		paths.KeyPath = consts.KUBE_TEST_CNI_TMP_KEY_DEFAULT_PATH
		return paths, nil
	}

	cert, err = GetLineFromYaml(string(confByte), "client-certificate")
	if err != nil {
		return nil, err
	}
	key, err = GetLineFromYaml(string(confByte), "client-key")
	if err != nil {
		return nil, err
	}

	err = utils2.FileCopy(cert, consts.KUBE_TEST_CNI_TMP_CERT_DEFAULT_PATH)
	if err != nil {
		return nil, err
	}
	err = utils2.FileCopy(key, consts.KUBE_TEST_CNI_TMP_KEY_DEFAULT_PATH)
	if err != nil {
		return nil, err
	}

	paths.CertPath = cert
	paths.KeyPath = key
	return paths, nil
}
