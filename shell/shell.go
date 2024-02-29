package shell

import (
	"bufio"
	"fmt"
	"golangsfs/in_out"
	"golangsfs/simdisk_procedure"

	"os"
	"os/exec"
	"strings"
)

func Shell(kind int, user_cmd []string, ch_in chan string, ch_out chan string) {
	// 后台运行标记位
	var Back_exec bool
	// 如果是后台运行就赋值传递channel
	if kind != 0 {
		in_out.CH_IN = ch_in
		in_out.CH_OUT = ch_out
	}

	// 说明有准备好的命令序列，在后台执行
	if len(user_cmd) != 0 {
		Back_exec = true
	}

	input := bufio.NewScanner(os.Stdin)
	index := 0
	// shell的循环
	for {
		// 获取当前工作目录
		now_working_dir, _ := os.Getwd()
		if !Back_exec {
			fmt.Print(now_working_dir + "-myshell-$ ")
		}

		var cmd string
		// 手动输入还是自动输入
		if kind == 0 {
			input.Scan()
			cmd = input.Text()
		} else {
			cmd = user_cmd[index]
			index++
		}

		// 去除前后多余空白符
		cmd = strings.TrimSpace(cmd)
		// 退出shell
		if cmd == "quit" {
			if !Back_exec {
				fmt.Print("退出shell……")
			}
			return
		}

		// 分割获取参数
		args := strings.Split(cmd, " ")

		// 进入simdisk
		if args[0] == "simdisk" {
			// 前端shell
			in_out.CH_IN = make(chan string, 1)
			in_out.CH_OUT = make(chan string, 1000)
			in_out.Ignore_print = 1
			// 后端go启动simdisk
			go simdisk_procedure.Simdisk()

			times := 0
			// simdisk的循环
			for {
				// 第一次要先等程序接收完信息
				if times != 0 {
					if kind == 0 {
						input.Scan()
						in_out.CH_IN <- input.Text()
					} else {
						in_out.CH_IN <- user_cmd[index]
						index++
					}
				}
				times++

				var message string
				var ok bool
				// 接收所有消息的循环
				for {
					// 从channel中接收消息
					message, ok = <-in_out.CH_OUT
					// 一些巧妙的逻辑，可以修复多进程阻塞恢复的信道bug
					// 读出多余消息
					if len(in_out.CH_OUT) != 0 && message == "finish" {
						// 要剩一个，防止下次循环无限阻塞读从而死锁
						for len(in_out.CH_OUT) != 1 {
							<-in_out.CH_OUT
						}
						continue
					}

					// simdisk退出 或是 消息传输完了
					if !ok || message == "finish" || message == "EXIT" {
						break
					} else {
						// 正常输出提示符或消息
						if strings.HasPrefix(message, "simdisk") {
							if !Back_exec {
								fmt.Print(message)
							}
						} else {
							if !Back_exec {
								fmt.Println(message)
							}
						}
					}
				}
				// break出这层循环用于结束simdisk程序
				if message == "EXIT" {
					break
				}
			}

		} else {
			// 必要的前两位
			// /C 是执行完不关闭
			cmd_list := []string{"cmd", "/C"}
			// 追加到前面
			for i := 0; i < len(args); i++ {
				cmd_list = append(cmd_list, args[i])
			}
			args = cmd_list

			// 调用cmd执行命令
			cmd_exec := exec.Command(args[0], args[1:]...)
			// 绑定标准输出和错误输出
			cmd_exec.Stdout = os.Stdout
			cmd_exec.Stderr = os.Stderr
			// 执行！
			err := cmd_exec.Run()
			if err != nil {
				if !Back_exec {
					fmt.Println("命令执行出错：", err)
				}

				continue
			}

			// 因为shell前面有工作目录提示符，所以需要特殊处理cd命令

			// 如果是特殊的cd\或是cd/
			if len(args) == 3 && (args[2] == "cd\\" || args[2] == "cd/") {
				os.Chdir("/")
			} else if args[2] == "cd" { // 普通cd
				// 试图分析参数中哪个是目录
				max_length := 0
				max_length_pos := 0

				tar_dir_path := ""
				breakFlag := false
				// 一般来说，参数中最长的是目录
				// 由于辅助参数最长就只有2个字符，所以特判长度为1或2的参数看是不是目录就可以了
				// 再长的话可以直接筛出来
				for i := 3; i < len(args); i++ {

					// 如果遇到这些参数，则cd到当前目录
					if args[i] == "." || args[i] == "./" || args[i] == ".\\" {
						tar_dir_path = args[i]
						breakFlag = true
						break
					}

					// 如果遇到一个字符且为斜杠或反斜杠的参数，则就是目录，且是当前盘符的根目录
					if args[i] == "\\" || args[i] == "/" {
						tar_dir_path = args[i]
						breakFlag = true
						break
					}

					// 如果遇到 .. 就cd到上级目录
					if args[i] == ".." {
						tar_dir_path = args[i]
						breakFlag = true
						break
					}

					// 不断更新最长标记
					if len(args[i]) > max_length {
						max_length = len(args[i])
						max_length_pos = i
					}
				}
				if breakFlag { // 说明是cd \ 或cd / 或cd当前目录
					os.Chdir(tar_dir_path)
				} else {
					// 如果不是
					// cd到最长参数对应目录
					os.Chdir(args[max_length_pos])
				}
			}
		}
	}
}
