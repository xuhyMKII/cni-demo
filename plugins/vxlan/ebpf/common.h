//下面代码是 eBPF 编程中的一些通用宏定义。
//这些宏定义通常用于简化 eBPF 代码编写和处理编译器警告。
//这些宏提供了对内存拷贝、调试输出、数据结构对齐和变量声明等功能的支持。

// __section 宏定义，用于将一个函数或变量放置在特定的 ELF section。
// 在 eBPF 程序中，这通常用于定义 eBPF maps 和 eBPF 函数。
#ifndef __section
# define __section(x)  __attribute__((section(x), used))
#endif

// bpf_memcpy 宏定义，使用内建的 __builtin_memcpy 函数实现内存拷贝。
#define bpf_memcpy __builtin_memcpy

// trace_printk 宏定义，用于在 eBPF 程序中进行调试输出。
// 通过将格式化字符串和参数传递给 bpf_trace_printk 函数，它将输出到内核的 trace buffer。
#define trace_printk(fmt, ...) do { \
	char _fmt[] = fmt; \
	bpf_trace_printk(_fmt, sizeof(_fmt), ##__VA_ARGS__); \
	} while (0)

// __packed 宏定义，用于指定一个数据结构在内存中的对齐方式。
// 使用 __packed 修饰的结构体将不会有任何对齐填充，从而节省内存空间。
#ifndef __packed
# define __packed		__attribute__((packed))
#endif

// __maybe_unused 宏定义，用于标记可能未使用的变量或函数，以避免编译器警告。
#ifndef __maybe_unused
# define __maybe_unused		__attribute__((__unused__))
#endif

// __section_maps_btf 宏定义，用于将 eBPF maps 放置在一个特定的 ELF section。
// 这通常用于与 BPF Type Format (BTF) 相关的功能，如 CO-RE (Compile Once, Run Everywhere)。
#ifndef __section_maps_btf
# define __section_maps_btf		__section(".maps")
#endif