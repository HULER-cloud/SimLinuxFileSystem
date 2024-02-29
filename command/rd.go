package command

import (
	"bufio"
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"golangsfs/zmap"
	"os"
	"strings"
)

func Rd(file *os.File, args []string) {

	target_dir := args[0]
	if target_dir == "/" {
		in_out.Out("根目录不能被删除！")
		return
	}

	now_dir_back := boot.Working_Directory
	now_dir_inode_back := boot.Working_INode
	now_dir_dentry_back := boot.Working_Dentry

	isSuccess := Cd(file, args, 0)
	if !isSuccess {
		in_out.Out("找不到目录！")
		return
	}

	// 能运行到这里说明目录可以找到
	// 以绝对路径记录要删除的目录，方便处理
	to_be_deleted_dir := boot.Working_Directory
	to_be_deleted_dir_inode := boot.Working_INode
	to_be_deleted_dir_dentry := boot.Working_Dentry

	boot.Working_Directory = now_dir_back
	boot.Working_INode = now_dir_inode_back
	boot.Working_Dentry = now_dir_dentry_back

	// 如果要删除的目录是当前工作目录或是其上级目录，拒绝删除
	if strings.HasPrefix(boot.Working_Directory, to_be_deleted_dir) {
		if len(boot.Working_Directory) == len(to_be_deleted_dir) {
			in_out.Out("不能删除当前所在目录！")
		} else {
			in_out.Out("不能删除当前目录的上级目录！")
		}
		return
	}

	// 如果要删除的目录就在当前工作目录中，或是其他分支的目录，允许删除
	now_dir_back = boot.Working_Directory
	now_dir_inode_back = boot.Working_INode
	now_dir_dentry_back = boot.Working_Dentry

	// 切到要删除的目录的上级目录执行删除
	Cd(file, args, 0)
	Cd(file, []string{".."}, 0)

	// 权限控制
	if boot.Working_User.Uid != "0" {
		target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
		if !RWX_JUDGE(target_dir_inode, "w") {
			in_out.Out("当前用户权限不足，无法在该目录删除目录！")
			return
		}
	}

	remove_dir(file, inode.Get_INode(file, to_be_deleted_dir_inode), inode.Get_Dentry(file, to_be_deleted_dir_dentry))

	boot.Working_Directory = now_dir_back
	boot.Working_INode = now_dir_inode_back
	boot.Working_Dentry = now_dir_dentry_back
}

func remove_dir(file *os.File, to_be_deleted_dir_inode inode.INode, to_be_deleted_dir_dentry inode.Dentry) {

	// 如果要删的是空目录，直接删除返回
	if len(to_be_deleted_dir_dentry.Content) == 0 {
		block_index := to_be_deleted_dir_inode.Zone[0]
		inode_index := to_be_deleted_dir_dentry.To_INode

		// 清文件块
		basic.Clean(file, boot.ZONE_START+block_index, boot.ZONE_START+block_index)
		// 清inode
		basic.Clean(file, boot.INODE_START+inode_index, boot.INODE_START+inode_index)
		// 清ZMap
		byte_index := block_index / 8
		bit_index := block_index % 8

		zmap.ZMap[byte_index] = ^(zmap.ZMap[byte_index] & (1 << (7 - bit_index)))
		zmap.Write_ZMap(file)
		zmap.Free_Block++
		zmap.Disk_size_remain++
		zmap.ZB.Free_Block = zmap.Free_Block
		zmap.ZB.Disk_size_remain = zmap.Disk_size_remain
		zmap.Write_ZMap_Block(file, zmap.ZB)

		// 清IMap
		basic.Clean(file, inode_index, inode_index)
		byte_index = inode_index / 8
		bit_index = inode_index % 8
		imap.IMap[byte_index] = ^(imap.IMap[byte_index] & (1 << (7 - bit_index)))
		imap.Write_IMap(file)
		imap.Free_INode++
		imap.IB.Free_INode++
		imap.Write_IMap_Block(file, imap.IB)

		// 处理上级目录冗余信息
		temp_working_dir_dentry := inode.Get_Dentry(file, to_be_deleted_dir_dentry.To_Father)
		for j := 0; j < len(temp_working_dir_dentry.Content); j++ {
			if temp_working_dir_dentry.Content[j] == inode_index {
				temp_working_dir_dentry.Content =
					append(temp_working_dir_dentry.Content[:j],
						temp_working_dir_dentry.Content[j+1:]...)
				break
			}
		}

		inode.Write_Dentry(file, temp_working_dir_dentry, to_be_deleted_dir_dentry.To_Father)
		// 更新内存中root相关
		if boot.Working_Directory == "/" {
			inode.Root_INode = inode.Get_INode(file, 0)
			inode.Root_Dentry = inode.Get_Dentry(file, 0)
		}
		return
	}

	// 往下是删除非空目录
	in_out.Out("要删除的目录非空，是否确定要删除？[Y/N]: ")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	if input.Text() == "Y" || input.Text() == "y" {
		for i := 0; i < len(to_be_deleted_dir_dentry.Content); i++ {
			temp_inside_inode :=
				inode.Get_INode(file, to_be_deleted_dir_dentry.Content[i])
			var temp_inside_dentry inode.Dentry
			if temp_inside_inode.Filetype == "dir" {
				temp_inside_dentry = inode.Get_Dentry(file, temp_inside_inode.Zone[0])
				remove_dir(file, temp_inside_inode, temp_inside_dentry)
			} else {
				fullname := temp_inside_inode.Filename
				if temp_inside_inode.Filetype != "" {
					fullname += ("." + temp_inside_inode.Filetype)
				}

				now_dir_back := boot.Working_Directory
				now_dir_inode_back := boot.Working_INode
				now_dir_dentry_back := boot.Working_Dentry

				// 切到要删除的文件的上级目录执行删除
				Cd(file, []string{to_be_deleted_dir_dentry.DName}, 0)
				//in_out.Out(to_be_deleted_dir_dentry.DName)
				//in_out.Out(fullname)
				Del(file, []string{fullname})

				boot.Working_Directory = now_dir_back
				boot.Working_INode = now_dir_inode_back
				boot.Working_Dentry = now_dir_dentry_back

			}
		}
		// 能走到这里说明内部的东西全删完了
		// 再递归一遍删除空的自己

		to_be_deleted_dir_inode = inode.Get_INode(file, to_be_deleted_dir_dentry.To_INode)
		to_be_deleted_dir_dentry = inode.Get_Dentry(file, to_be_deleted_dir_inode.Zone[0])
		//in_out.Out(to_be_deleted_dir_dentry.Content)
		//temp_father_dir_inode := inode.Get_INode(file, to_be_deleted_dir_dentry.To_Father)
		//temp_father_dir_dentry := inode.Get_Dentry(file, temp_father_dir_inode.Zone[0])
		remove_dir(file, to_be_deleted_dir_inode, to_be_deleted_dir_dentry)

		// 处理上级目录冗余信息
		temp_working_dir_dentry := inode.Get_Dentry(file, boot.Working_Dentry)
		for j := 0; j < len(temp_working_dir_dentry.Content); j++ {
			if temp_working_dir_dentry.Content[j] == to_be_deleted_dir_dentry.To_INode {
				temp_working_dir_dentry.Content =
					append(temp_working_dir_dentry.Content[:j],
						temp_working_dir_dentry.Content[j+1:]...)
				break
			}
		}
		// 更新内存中root相关
		if boot.Working_Directory == "/" {
			inode.Root_INode = inode.Get_INode(file, 0)
			inode.Root_Dentry = inode.Get_Dentry(file, 0)
		}
		inode.Write_Dentry(file, temp_working_dir_dentry, boot.Working_Dentry)

	}
}
