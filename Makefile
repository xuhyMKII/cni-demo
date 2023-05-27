build_ebpf:
	clang -g  -O2 -emit-llvm -c ./plugins/vxlan/ebpf/vxlan_egress.c -o - | llc -march=bpf -filetype=obj -o vxlan_egress.o
	clang -g  -O2 -emit-llvm -c ./plugins/vxlan/ebpf/vxlan_ingress.c -o - | llc -march=bpf -filetype=obj -o vxlan_ingress.o
	clang -g  -O2 -emit-llvm -c ./plugins/vxlan/ebpf/veth_ingress.c -o - | llc -march=bpf -filetype=obj -o veth_ingress.o
	mv veth_ingress.o /opt/cni-demo/
	mv vxlan_ingress.o /opt/cni-demo/
	mv vxlan_egress.o /opt/cni-demo/

build_main:
	go build main.go

build:
	go build .
	make build_ebpf