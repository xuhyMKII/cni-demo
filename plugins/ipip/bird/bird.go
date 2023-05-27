package bird

import (
	"cni-demo/consts"
	"cni-demo/tools/utils"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// StartBirdDaemon 用于启动 BIRD 守护进程。
// 这个函数会先检查 BIRD 配置文件是否存在，然后确认 BIRD 守护进程是否已经在运行。
// 如果已经在运行，函数会直接返回当前的进程 ID (PID)。
// 如果没有运行，函数会启动一个新的 BIRD 守护进程，并将其 PID 写入一个文件，
// 以便后续可以方便地找到并管理这个进程。
//
// 参数:
//   - configPath: BIRD 配置文件的路径。
//
// 返回值:
//   - int: BIRD 守护进程的 PID。
//   - error: 如果在启动 BIRD 守护进程的过程中遇到错误，返回一个 error 对象
func StartBirdDaemon(configPath string) (int, error) {
	if !utils.FileIsExisted(configPath) {
		return -1, fmt.Errorf("the config path %s not exist", configPath)
	}

	// 先看 bird deamon 这个路径是否存在
	if utils.PathExists(consts.KUBE_TEST_CNI_DEFAULT_BIRD_DEAMON_PATH) {
		// 如果该路径存在
		pid, err := utils.ReadContentFromFile(consts.KUBE_TEST_CNI_DEFAULT_BIRD_DEAMON_PATH)
		if err != nil {
			return -1, err
		}
		// 尝试读出里头的 pid, 然后看这个 pid 当前是不是真的在运行
		if utils.FileIsExisted(fmt.Sprintf("/proc/%s", pid)) {
			// 说明当前 host 上的 bird 正在运行可以直接返回
			return strconv.Atoi(pid)
		} else {
			// 说明当前 host 上的 bird 已经退出了, 那就删掉这个文件
			utils.DeleteFile(consts.KUBE_TEST_CNI_DEFAULT_BIRD_DEAMON_PATH)
		}
	}

	cmd := exec.Command(
		"/opt/cni-demo/bird",
		"-R",
		"-s",
		"/var/run/bird.ctl",
		"-d",
		"-c",
		configPath,
	)
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		return -1, err
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	utils.CreateFile(consts.KUBE_TEST_CNI_DEFAULT_BIRD_DEAMON_PATH, ([]byte)(pid), 0766)
	return cmd.Process.Pid, nil
}
