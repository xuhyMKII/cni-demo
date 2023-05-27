# 测试 CNI 插件

本文档将介绍如何在 Kubernetes 集群中测试不同模式的 CNI 插件。以下内容将分别介绍 IPIP 模式、VxLAN 模式、IPVlan & MACVlan 模式以及 Host-gw 模式的测试方法。

## 准备工作

确保您的 Kubernetes 环境是干净的，没有安装任何网络插件。

## IPIP 模式测试

1. 在 `/etc/cni/net.d/` 目录下创建一个以 `.conf` 结尾的文件，输入以下配置：
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>json</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-json">{
     &#34;cniVersion&#34;: &#34;0.3.0&#34;,
     &#34;name&#34;: &#34;testcni&#34;,
     &#34;type&#34;: &#34;testcni&#34;,
     &#34;mode&#34;: &#34;ipip&#34;,
     &#34;subnet&#34;: &#34;10.244.0.0/16&#34;
   }
   </code></div></div></pre>
2. 在项目根目录执行 `make build_main`，生成一个名为 `main` 的二进制文件。
3. 克隆 [Calico BIRD 项目](https://github.com/projectcalico/bird)，编译 Calico 的 BIRD 二进制文件。
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>bash</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-bash">ARCH=&lt;你的计算机架构&gt; ./build.sh
   </code></div></div></pre>
4. 创建 `/opt/testcni` 目录，将第 3 步生成的 BIRD 二进制文件拷贝到该目录下。
5. 将第 2 步生成的 `main` 二进制文件拷贝到 `/opt/cni/bin/testcni` 目录下。
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>bash</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-bash">mv main /opt/cni/bin/testcni
   </code></div></div></pre>

## VxLAN 模式测试

1. 在 `/etc/cni/net.d/` 目录下创建一个以 `.conf` 结尾的文件，输入以下配置：
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>json</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-json">{
     &#34;cniVersion&#34;: &#34;0.3.0&#34;,
     &#34;name&#34;: &#34;testcni&#34;,
     &#34;type&#34;: &#34;testcni&#34;,
     &#34;mode&#34;: &#34;vxlan&#34;,
     &#34;subnet&#34;: &#34;10.244.0.0&#34;
   }
   </code></div></div></pre>
2. 在项目根目录执行 `make build`，生成一个名为 `testcni` 的二进制文件以及三个 eBPF 文件。
3. 将第 2 步生成的 eBPF 文件拷贝到 `/opt/testcni/` 目录下（如果目录不存在，手动创建）。
4. 将第 2 步生成的 `testcni` 二进制文件拷贝到 `/opt/cni/bin` 目录下。

## IPVlan & MACVlan 模式测试

1. 在每个节点的 `/etc/cni/net.d/` 目录下创建一个以 `.conf` 结尾的文件，输入以下配置。请注意修改 `subnet` 和 `ipam` 中的 `range`，以适应您的实际环境，同时确保每个节点的 `range` 配置不同。
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>json</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-json">{
     &#34;cniVersion&#34;: &#34;0.3.0&#34;,
     &#34;name&#34;: &#34;testcni&#34;,
     &#34;type&#34;: &#34;testcni&#34;,
     &#34;mode&#34;: &#34;ipvlan&#34;,
     &#34;subnet&#34;: &#34;192.168.64.0/24&#34;,
     &#34;ipam&#34;: {
       &#34;rangeStart&#34;: &#34;192.168.64.90&#34;,
       &#34;rangeEnd&#34;: &#34;192.168.64.100&#34;
     }
   }
   </code></div></div></pre>
2. 在项目根目录执行 `make build_main`，生成一个名为 `main` 的二进制文件。
3. 将第 2 步生成的 `main` 二进制文件拷贝到 `/opt/cni/bin/testcni` 目录下。
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>bash</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-bash">mv main /opt/cni/bin/testcni
   </code></div></div></pre>
4. 在集群中部署测试容器，如 `busybox`，并观察集群中的 Pod 状态。如果 Pods 正常启动并运行，说明 IPVlan 或 MACVlan 模式的 CNI 插件已经成功安装并配置。

## Host-gw 模式测试

1. 在 `/etc/cni/net.d/` 目录下创建一个以 `.conf` 结尾的文件，输入以下配置：
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>json</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-json">{
     &#34;cniVersion&#34;: &#34;0.3.0&#34;,
     &#34;name&#34;: &#34;testcni&#34;,
     &#34;type&#34;: &#34;testcni&#34;,
     &#34;bridge&#34;: &#34;testcni0&#34;,
     &#34;subnet&#34;: &#34;10.244.0.0/16&#34;
   }
   </code></div></div></pre>
2. 修改 `/etcd/client.go` 文件中的 IP 地址，将其更改为您集群的 etcd 地址。
3. 在项目根目录执行 `go build main.go`，生成一个名为 `main` 的二进制文件。
4. 将第 3 步生成的 `main` 二进制文件拷贝到 `/opt/cni/bin/testcni` 目录下。
   <pre class=""><div class="bg-black rounded-md mb-4"><div class="flex items-center relative text-gray-200 bg-gray-800 px-4 py-2 text-xs font-sans justify-between rounded-t-md"><span>bash</span><button class="flex ml-auto gap-2"><svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg"><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path><rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect></svg>Copy code</button></div><div class="p-4 overflow-y-auto"><code class="!whitespace-pre hljs language-bash">mv main /opt/cni/bin/testcni
   </code></div></div></pre>
5. 在每个集群主机上重复步骤 1 至 4。
6. 使用 `kubectl apply -f test-busybox.yaml` 部署一个测试容器，如 `busybox`。
7. 观察集群中的 Pod 状态。如果 Pods 正常启动并运行，说明 host-gw 模式的 CNI 插件已经成功安装并配置。






