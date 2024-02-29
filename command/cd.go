package command

import (
	"golangsfs/boot"
	"golangsfs/in_out"
	"golangsfs/inode"
	"os"
	"strings"
)

func Cd(file *os.File, args []string, kind int) bool {

	// 因为无判断的cd需要用到，所以判断权限的逻辑单独封装在if中，需要的时候调用
	// kind=0是有判断的
	// root用户可以无视权限控制
	if kind == 0 && boot.Working_User.Uid != "0" {
		target_dir_inode := Get_target_dir_inode(file, args[0])
		if !RWX_JUDGE(target_dir_inode, "cd") {
			in_out.Out("当前用户权限不足，无法切换到该目录！")
			return false
		}
	}

	target_dir := args[0] // 只获取cd后面的第一个参数就够了

	// 特判一下，如果切到根目录直接返回即可
	if target_dir == "/" {
		boot.Working_Directory = "/"
		boot.Working_INode = 0
		boot.Working_Dentry = 0
		return true
	}
	// 就在本目录的也是直接返回
	if target_dir == "." || target_dir == "./" {
		return true
	}

	if strings.HasPrefix(target_dir, "/") { // 绝对路径

		// 去除最后的多余斜杠
		end := len(target_dir)
		if strings.HasSuffix(target_dir, "/") {
			end--
		}
		// 获取路径分量
		path := strings.Split(target_dir[1:end], "/")

		// 追踪目录，0代表绝对路径，1代表相对路径
		inode_index, dentry_index, found := dir_trace(file, path, 0)
		// 找到了就修改当前工作目录的信息
		if found {
			boot.Working_Directory = target_dir[:end]
			boot.Working_INode = inode_index
			boot.Working_Dentry = dentry_index
		} else {
			return false
		}

	} else { // 相对路径

		// 删除无用前缀
		start := 0
		if strings.HasPrefix(target_dir, "./") {
			start = 2
		}
		// 删除无用后缀
		end := len(target_dir)
		if strings.HasSuffix(target_dir, "/") {
			end--
		}
		// 获取路径分量
		path := strings.Split(target_dir[start:end], "/")

		inode_index, dentry_index, found := dir_trace(file, path, 1)
		if found {
			boot.Working_INode = inode_index
			boot.Working_Dentry = dentry_index
			// 相对路径没法直接赋值给绝对路径类型值使用，只能这样间接查询
			boot.Working_Directory = inode.Get_Dentry(file, dentry_index).DName

		} else {
			return false
		}
	}
	return true
}

func dir_trace(file *os.File, path []string, kind int) (inode_index int, dentry_index int, mark bool) {
	// 记录工作目录信息
	inode_index = boot.Working_INode
	dentry_index = boot.Working_Dentry

	// 临时变量用来获取inode和dentry
	var temp_working_inode inode.INode
	var temp_working_dentry inode.Dentry

	// 如果是cd绝对路径就先切换到根目录
	if kind == 0 {
		temp_working_inode = inode.Root_INode
		temp_working_dentry = inode.Root_Dentry
	} else { // 否则获取当前工作目录
		temp_working_inode = inode.Get_INode(file, boot.Working_INode)
		temp_working_dentry = inode.Get_Dentry(file, boot.Working_Dentry)
	}

	// 对于路径上的每一个环节
	for i := 0; i < len(path); i++ {
		// 解决相对目录找同级的问题
		if path[i] == "." {
			continue
		}

		// 解决相对目录找上级的问题
		if path[i] == ".." {
			// 找到上级的inode
			inode_index = temp_working_dentry.To_Father
			temp_working_inode = inode.Get_INode(file, inode_index)
			dentry_index = temp_working_inode.Zone[0]
			temp_working_dentry = inode.Get_Dentry(file, dentry_index)

			continue
		}

		// 获取到当前目录所含项的inode坐标列表
		temp_content := temp_working_dentry.Content

		// 没有找到的flag
		notfound := true
		// 接着循环检查每一项
		for j := 0; j < len(temp_content); j++ {
			// 获取到每一项的inode
			temp_inode := inode.Get_INode(file, temp_content[j])

			// 如果找到了和当前环节同名的目录或文件，进一步检查
			if path[i] == temp_inode.Filename {
				// 如果找到的是文件，continue
				if temp_inode.Filetype != "dir" {
					continue
				} else { // 如果找到了同名目录
					// 未找到标记更改为false
					notfound = false
					// 将当前工作目录的inode和dentry修改为找到的目录
					inode_index = temp_content[j]
					temp_working_inode = inode.Get_INode(file, inode_index)

					dentry_index = temp_working_inode.Zone[0]
					temp_working_dentry = inode.Get_Dentry(file, dentry_index)
				}
			}
		}
		// 如果遍历完inode坐标列表还没有找到，就说明目录是错的
		if notfound {
			return -1, -1, false
		}
	}
	return inode_index, dentry_index, true
}
