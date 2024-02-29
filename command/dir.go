package command

import (
	"golangsfs/boot"
	"golangsfs/in_out"
	"golangsfs/inode"
	"os"
	"strconv"
)

func Dir(file *os.File, args []string) {

	// 权限控制
	if boot.Working_User.Uid != "0" {
		var target_dir_inode inode.INode
		if len(args) == 0 {
			target_dir_inode = inode.Get_INode(file, boot.Working_INode)
		} else {
			target_dir_inode = Get_target_dir_inode(file, args[0])
		}
		if !RWX_JUDGE(target_dir_inode, "dir") {
			in_out.Out("当前用户权限不足，无法查看该目录信息！")
			return
		}
	}

	// 先处理只展示本目录，且不递归的情况
	if len(args) == 0 {
		in_out.Out("目录 " + boot.Working_Directory + " 内容如下：")
		inode_list := inode.Get_Dentry(file, boot.Working_Dentry).Content
		if len(inode_list) == 0 {
			in_out.Out("目录 " + boot.Working_Directory + " 为空！")
			in_out.Out("")
		}
		// 遍历每一项输出相关信息
		for i := 0; i < len(inode_list); i++ {
			temp_inode := inode.Get_INode(file, inode_list[i])
			temp_filename := temp_inode.Filename
			temp_physics_address := (temp_inode.Zone[0] + boot.ZONE_START) * 1024

			// 控制位转换写法
			temp_control := ""
			for i := 0; i < 3; i++ {
				if temp_inode.Control[i]/4 == 1 {
					temp_control += "r"
				} else {
					temp_control += "-"
				}
				if temp_inode.Control[i]/2%2 == 1 {
					temp_control += "w"
				} else {
					temp_control += "-"
				}
				if temp_inode.Control[i]%2 == 1 {
					temp_control += "x"
				} else {
					temp_control += "-"
				}
			}

			temp_filesize := temp_inode.File_size
			if temp_inode.Filetype == "dir" {
				temp_control = "d" + temp_control
				in_out.Out(temp_filename + " " + strconv.Itoa(temp_physics_address) + " <DIR>" + "\t" + temp_control)
			} else {
				temp_control = "-" + temp_control
				if temp_physics_address == 26214400 {
					temp_physics_address = -1
				}
				if temp_inode.Filetype == "" {
					in_out.Out(temp_filename + " " + strconv.Itoa(temp_physics_address) + "\t" + strconv.Itoa(temp_filesize) + " " + temp_control)
				} else {
					in_out.Out(temp_filename + "." + temp_inode.Filetype + " " + strconv.Itoa(temp_physics_address) + "\t" + strconv.Itoa(temp_filesize) + "KB " + temp_control)
				}

			}
			// 在最后输出空行，看起来美观一点
			if i == len(inode_list)-1 {
				in_out.Out("")
			}
		}

	} else if len(args) == 1 {
		// 如果参数是 /s
		if args[0] == "/s" {
			dir_s(file)
		} else { // 不带/s，只有目标目录
			target_dir := []string{args[0]}
			now_dir_back := boot.Working_Directory
			now_dir_inode_back := boot.Working_INode
			now_dir_dentry_back := boot.Working_Dentry

			isSuccess := Cd(file, target_dir, 0)
			if !isSuccess {
				in_out.Out("找不到目录！")
				in_out.Out("")

			}
			Dir(file, []string{})

			boot.Working_Directory = now_dir_back
			boot.Working_INode = now_dir_inode_back
			boot.Working_Dentry = now_dir_dentry_back

		}
	} else {
		var target_dir []string
		if args[0] == "/s" {
			target_dir = []string{args[1]}
		} else {
			target_dir = []string{args[0]}
		}

		now_dir_back := boot.Working_Directory
		now_dir_inode_back := boot.Working_INode
		now_dir_dentry_back := boot.Working_Dentry

		isSuccess := Cd(file, target_dir, 0)
		if !isSuccess {
			in_out.Out("找不到目录！")
			in_out.Out("")
		}
		dir_s(file)

		boot.Working_Directory = now_dir_back
		boot.Working_INode = now_dir_inode_back
		boot.Working_Dentry = now_dir_dentry_back
	}

}

// 递归展开目录的专属函数
func dir_s(file *os.File) {
	inode_list := inode.Get_Dentry(file, boot.Working_Dentry).Content
	var dir_list []int
	// 遍历，看目录中有多少需要递归展开的子目录
	for i := 0; i < len(inode_list); i++ {
		temp_inode := inode.Get_INode(file, inode_list[i])
		temp_filetype := temp_inode.Filetype
		if temp_filetype == "dir" {
			dir_list = append(dir_list, inode_list[i])
		}
	}
	// 解决本目录的dir
	Dir(file, []string{})
	for i := 0; i < len(dir_list); i++ {
		// 获取到路径
		temp_inode := inode.Get_INode(file, dir_list[i])
		temp_dentry := inode.Get_Dentry(file, temp_inode.Zone[0])

		target_dir := []string{temp_dentry.DName}

		now_dir_back := boot.Working_Directory
		now_dir_inode_back := boot.Working_INode
		now_dir_dentry_back := boot.Working_Dentry

		isSuccess := Cd(file, target_dir, 0)
		if !isSuccess {
			in_out.Out("找不到目录！")
			in_out.Out("")
		}
		// 最后递归展开cd过去的目录（刚才的子目录）
		Dir(file, []string{"/s"})

		boot.Working_Directory = now_dir_back
		boot.Working_INode = now_dir_inode_back
		boot.Working_Dentry = now_dir_dentry_back

	}
}
