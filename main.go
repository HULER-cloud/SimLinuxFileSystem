package main

import (
	"bufio"
	"fmt"
	"golangsfs/shell"
	"golangsfs/simdisk_procedure"
	"os"
)

func main() {

	fmt.Println("请选择直接操作(输入1)、shell远程操作(输入2)，或是模拟多进程并发操作(输入3)")
	fmt.Println("输入其他字符以退出程序")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	switch input.Text() {
	case "1":
		simdisk_procedure.Simdisk()
	case "2":
		shell.Shell(0, []string{}, make(chan string), make(chan string))
	case "3":
		// 打算模拟两个进程并发操作，一个进程对应一个用户
		// 分别是simdisk中的1号和2号普通用户，所属同组

		// 命令序列，后台shell自动执行用
		cmd1 := []string{
			"simdisk",
			"2",
			"cd home/second",
			"newfile back.txt",
			"dir",
			"EXIT",
			"quit",
		}
		// 创建两进两出4个channel
		chan_in1 := make(chan string, 1)
		chan_out1 := make(chan string, 1000)
		chan_in2 := make(chan string, 1)
		chan_out2 := make(chan string, 1000)

		// 启动后台shell
		go shell.Shell(1, cmd1, chan_in2, chan_out2)
		// 启动前台shell
		shell.Shell(0, []string{}, chan_in1, chan_out1)

	default:
		fmt.Println("退出程序……")
	}

}
