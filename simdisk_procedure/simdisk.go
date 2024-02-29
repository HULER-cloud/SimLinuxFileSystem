package simdisk_procedure

import (
	"bufio"
	"golangsfs/boot"
	"golangsfs/command"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"golangsfs/super"
	"golangsfs/writing"
	"golangsfs/zmap"
	"os"
	"strings"
	"time"
)

func Simdisk() {
	// 启动状态自检，如果disk不存在则创建
	init_check()

	start_initialize()

	// 下面是主要的功能逻辑

	in_out.Out("欢迎使用simdisk磁盘管理系统！\n请选择登录用户UID:")

	for i := 0; i < len(boot.User_list); i++ {
		in_out.Out(boot.User_list[i].Uid + " " + boot.User_list[i].Username)
	}

	in_out.Out("finish")

	input := bufio.NewScanner(os.Stdin)
	var cmd string
	if in_out.Ignore_print == 1 {
		cmd = <-in_out.CH_IN
	} else {
		input.Scan()
		cmd = input.Text()
	}

	breakFlag := false
	for _, user := range boot.User_list {
		if cmd == user.Uid {
			in_out.Out("以用户 " + user.Username + " 身份登录成功！")
			boot.Working_User = user
			breakFlag = true
			break
		}
	}
	if !breakFlag {
		in_out.Out(cmd + " 号用户不存在！系统退出！")
		return
	}

	file, _ := os.OpenFile(boot.DISK_PATH_WHOLE, os.O_RDWR, 0777)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			in_out.Out("磁盘关闭异常！错误信息：" + err.Error())
			return
		}
	}(file) // 可能不需要，先留着

	// 无限循环接收命令（和参数），直到键入EXIT退出
	for {

		in_out.Out("simdisk- " + boot.Working_Directory + " ->>> ")
		in_out.Out("finish")
		// 读入命令
		if in_out.Ignore_print == 1 {
			cmd = <-in_out.CH_IN
		} else {
			input.Scan()
			cmd = input.Text()
		}
		// 空命令直接跳过本次循环
		if cmd == "" {
			continue
		}

		if cmd != "EXIT" {

			recv_command := strings.TrimSpace(cmd)
			// 捕获使用的命令
			command_and_args := strings.Split(recv_command, " ")
			switch command_and_args[0] {
			case "info":
				command.Info()
			case "cd":
				// 切片可以超出索引没有问题，不会崩溃
				isSuccess := command.Cd(file, command_and_args[1:], 0)
				if !isSuccess {
					in_out.Out("找不到目录！")
				}
			case "dir":
				command.Dir(file, command_and_args[1:])
			case "md":
				// 互斥写测试
				writing.Writing_Test(file)
				// 必要的更新
				command.Update(file)
				// 写前设1
				writing.To_Writing(file)
				command.Md(file, command_and_args[1:])
				// 邂逅恢复1
				writing.To_Not_Writing(file)
			case "rd":
				writing.Writing_Test(file)
				command.Update(file)
				writing.To_Writing(file)
				command.Rd(file, command_and_args[1:])
				writing.To_Not_Writing(file)
			case "newfile":
				writing.Writing_Test(file)
				command.Update(file)
				writing.To_Writing(file)
				command.Newfile(file, command_and_args[1:])
				writing.To_Not_Writing(file)
			case "cat":
				file_data := command.Cat(file, command_and_args[1:])
				// 在外界输出
				if len(file_data) != 0 {
					in_out.Out(string(file_data))
				}
			case "copy":
				writing.Writing_Test(file)
				command.Update(file)
				writing.To_Writing(file)
				command.Copy(file, command_and_args[1:])
				writing.To_Not_Writing(file)
			case "del":
				command.Del(file, command_and_args[1:])
			case "check":
				writing.Writing_Test(file)
				command.Update(file)
				writing.To_Writing(file)
				command.Check(file)
				writing.To_Not_Writing(file)

			default:
				in_out.Out("不支持的命令，请重新输入！")
			}
		} else {
			// 退出前可能需要写入一些东西

			in_out.Out("退出simdisk系统……")
			if in_out.Ignore_print == 1 {
				in_out.Out("EXIT")
			}

			return
		}

	}
}

// 启动前自检
func init_check() {
	in_out.Out("正在执行自检程序……")
	file, err := os.OpenFile(boot.DISK_PATH_WHOLE, os.O_RDWR, 0777)
	if err != nil {
		if os.IsNotExist(err) {
			in_out.Out("disk不存在，现在创建……")
			disk_init_size := make([]byte, boot.DISK_SIZE_BYTE)
			disk, err := os.Create(boot.DISK_PATH_WHOLE)
			if err != nil {
				in_out.Out("disk创建失败！错误信息：" + err.Error())
				return
			}
			// 返回值写入空间大小被忽略，这个是已知的
			_, err = disk.Write(disk_init_size)
			if err != nil {
				in_out.Out("disk空间申请失败！错误信息：" + err.Error())
				return
			}
			in_out.Out("disk创建成功！")
			// 新建成功时把disk指针写回file中，防止下面关闭出错
			file = disk
			// 这里放disk的初始化程序（从0开始）
			construct_initialize()
		} else {
			in_out.Out("disk自检失败！错误信息：" + err.Error())
			return
		}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			in_out.Out("检查程序退出异常！错误信息：" + err.Error())
			return
		}
		in_out.Out("自检结束，进入simdisk程序……")
	}(file)
}

// 由于初始化过程，除了时间其他全是固定的，有必要以写死的方式呈现，不然太尼玛乱了
func construct_initialize() {
	file, _ := os.OpenFile(boot.DISK_PATH_WHOLE, os.O_RDWR, 0777)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			in_out.Out("磁盘关闭异常！错误信息：" + err.Error())
			return
		}
	}(file)

	// 首先是boot块的东西
	// root用户和root组
	boot.User_list[0] = boot.Root_user
	boot.Group_list[0] = boot.Root_group
	// 普通用户和普通组
	boot.User_list[1] = boot.Ordinary_user_default
	boot.User_list[2] = boot.Ordinary_user_second
	boot.User_list[3] = boot.Ordinary_user_outside
	boot.Group_list[1] = boot.Oridinary_group
	boot.Group_list[2] = boot.Another_group

	// 设置Boot_Block的属性，并写入磁盘
	boot.BB = boot.Boot_Block{
		Next_UID:   4,
		Next_GID:   3,
		User_list:  boot.User_list,
		Group_list: boot.Group_list,
	}
	boot.Write_Boot_Block(file, boot.BB)

	// 接下来是Super_Block
	// 为了直观展示，不写常量定义，直接写数字了
	super.SB = super.Super_Block{
		Max_file_size: 65800,
		IMap_blocks:   4,
		ZMap_blocks:   10,
		INodes:        25584,
		Blocks:        76800,
	}
	super.Write_Super_Block(file, super.SB)

	// imap块相关
	// 初始化全局IMap
	// IMap的第一个字节设置成252=11111100
	imap.IMap = make([]byte, (boot.ZMAP_START-boot.IMAP_START)*1024)
	imap.IMap[0] = 252
	imap.Write_IMap(file)
	// 调整空闲inode情况
	// 额外减6，在构造初始化过程中我们一共会用到6个inode
	// Last_position依然是0，不用改变
	imap.Free_INode = boot.ZONE_START - boot.INODE_START - 6
	// 写入磁盘
	ib := imap.IMap_Block{
		Free_INode:    imap.Free_INode,
		Last_position: imap.Last_position,
	}
	imap.Write_IMap_Block(file, ib)

	// zmap块相关，一些额外操作同imap
	// 初始化全局ZMap
	zmap.ZMap = make([]byte, (boot.INODE_START-boot.ZMAP_START)*1024)
	zmap.ZMap[0] = 252
	zmap.Write_ZMap(file)
	// 调整磁盘最大剩余容量
	zmap.Free_Block = boot.DISK_SIZE_BLOCK - boot.ZONE_START - 6
	zmap.Disk_size_remain = zmap.Free_Block * 1024
	// 写入磁盘
	zb := zmap.ZMap_Block{
		Free_Block:       zmap.Free_Block,
		Disk_size_remain: zmap.Disk_size_remain,
		Last_position:    zmap.Last_position,
	}
	zmap.Write_ZMap_Block(file, zb)

	// 分配根目录inode
	curr_time := time.Now()
	inode.Root_INode = inode.INode{
		Filename:    "/",
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     [3]int8{7, 5, 5},
		Owner_id:    boot.Root_user.Uid,
		Group_id:    boot.Root_user.Gid,
		Zone:        [10]int{0},
	}
	inode.Write_INode(file, inode.Root_INode, 0)
	// 分配根目录文件块
	inode.Root_Dentry = inode.Dentry{
		To_INode:    0,
		To_Father:   0,
		DName:       "/",
		DName_Short: "/",
		Content:     []int{1, 2},
	}
	inode.Write_Dentry(file, inode.Root_Dentry, 0)

	// 分配/root的inode
	curr_time = time.Now()
	root_user_dir_inode := inode.INode{
		Filename:    "root",
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     [3]int8{7, 5, 5},
		Owner_id:    boot.Root_user.Uid,
		Group_id:    boot.Root_user.Gid,
		Zone:        [10]int{1},
	}
	inode.Write_INode(file, root_user_dir_inode, 1)
	// 分配/root的文件块
	root_user_dir_dentry := inode.Dentry{
		To_INode:    1,
		To_Father:   0,
		DName:       "/root",
		DName_Short: "root",
		Content:     make([]int, 0),
	}
	inode.Write_Dentry(file, root_user_dir_dentry, 1)

	// 分配/home的inode
	curr_time = time.Now()
	home_dir_inode := inode.INode{
		Filename:    "home",
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     [3]int8{7, 5, 5},
		Owner_id:    "0",
		Group_id:    "0",
		Zone:        [10]int{2},
	}
	inode.Write_INode(file, home_dir_inode, 2)
	// 分配/home的文件块
	home_dir_dentry := inode.Dentry{
		To_INode:    2,
		To_Father:   0,
		DName:       "/home",
		DName_Short: "home",
		Content:     []int{3, 4, 5},
	}
	inode.Write_Dentry(file, home_dir_dentry, 2)

	// 分配/home/default的inode
	curr_time = time.Now()
	default_user_dir_inode := inode.INode{
		Filename:    "default",
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     [3]int8{7, 5, 5},
		Owner_id:    "1",
		Group_id:    "1",
		Zone:        [10]int{3},
	}
	inode.Write_INode(file, default_user_dir_inode, 3)
	// 分配/home/default_user的文件块
	default_user_dir_dentry := inode.Dentry{
		To_INode:    3,
		To_Father:   2,
		DName:       "/home/default",
		DName_Short: "default",
		Content:     make([]int, 0),
	}
	inode.Write_Dentry(file, default_user_dir_dentry, 3)

	// 分配/home/second的inode
	curr_time = time.Now()
	second_user_dir_inode := inode.INode{
		Filename:    "second",
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     [3]int8{7, 5, 5},
		Owner_id:    "2",
		Group_id:    "1",
		Zone:        [10]int{4},
	}
	inode.Write_INode(file, second_user_dir_inode, 4)
	// 分配/home/second的文件块
	second_user_dir_dentry := inode.Dentry{
		To_INode:    3,
		To_Father:   2,
		DName:       "/home/second",
		DName_Short: "second",
		Content:     make([]int, 0),
	}
	inode.Write_Dentry(file, second_user_dir_dentry, 4)

	// 分配/home/outside的inode
	curr_time = time.Now()
	outside_user_dir_inode := inode.INode{
		Filename:    "outside",
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     [3]int8{7, 5, 5},
		Owner_id:    "2",
		Group_id:    "1",
		Zone:        [10]int{5},
	}
	inode.Write_INode(file, outside_user_dir_inode, 5)
	// 分配/home/outside的文件块
	outside_user_dir_dentry := inode.Dentry{
		To_INode:    3,
		To_Father:   2,
		DName:       "/home/outside",
		DName_Short: "outside",
		Content:     make([]int, 0),
	}
	inode.Write_Dentry(file, outside_user_dir_dentry, 5)

}

// 启动时初始化的函数
func start_initialize() {
	file, _ := os.OpenFile(boot.DISK_PATH_WHOLE, os.O_RDWR, 0777)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			in_out.Out("磁盘关闭异常！错误信息：" + err.Error())
			return
		}
	}(file)

	// 为了在已有simdisk的情况下直接初始化启动
	// 在construct_initialize()里面获取到的全局变量要重新get
	// 即使这些全局结构体的值没有改变

	// boot块
	// 工作目录相关东西和Boot_Block要获取到
	// 避免循环import，boot的这两个属性以编号的形式给出，而非结构体
	boot.Working_Directory = "/"
	boot.Working_INode = 0
	boot.Working_Dentry = 0
	boot.BB = boot.Get_Boot_Block(file)
	boot.User_list = boot.BB.User_list
	boot.Group_list = boot.BB.Group_list

	// 获取超级块
	super.SB = super.Get_Super_Block(file)

	// imap块
	imap.Get_IMap(file)
	imap.IB = imap.Get_IMap_Block(file)
	imap.Free_INode = imap.IB.Free_INode
	imap.Last_position = imap.IB.Last_position

	// zmap块
	zmap.Get_ZMap(file)
	zmap.ZB = zmap.Get_ZMap_Block(file)
	zmap.Free_Block = zmap.ZB.Free_Block
	zmap.Last_position = zmap.ZB.Last_position
	zmap.Disk_size_remain = zmap.ZB.Disk_size_remain

	// 把根目录的这两个全局结构体获取到之后，后面确实没东西了
	inode.Root_INode = inode.Get_INode(file, 0)
	inode.Root_Dentry = inode.Get_Dentry(file, 0)
}
