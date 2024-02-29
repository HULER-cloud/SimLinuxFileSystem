package command

import (
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"golangsfs/zmap"
	"os"
	"time"
)

func Md(file *os.File, args []string) {
	index := 0
	// 如果两个参数，那么要创建的目录就是1号
	if len(args) == 2 {
		index = 1
	}
	// 拒绝创建多级目录
	for i := 0; i < len(args[index]); i++ {
		if args[index][i] == '/' {
			in_out.Out("不支持创建多级目录！")
			return
		}
	}

	// 只有一个参数说明在当前目录创建
	if len(args) == 1 {
		// 权限控制
		if boot.Working_User.Uid != "0" {
			target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
			if !RWX_JUDGE(target_dir_inode, "w") {
				in_out.Out("当前用户权限不足，无法在该目录新建目录！")
				return
			}
		}

		mk_dir(file, args[0])
	} else { // 那就cd到其他目录创建

		now_dir_back := boot.Working_Directory
		now_dir_inode_back := boot.Working_INode
		now_dir_dentry_back := boot.Working_Dentry

		// 权限控制
		if boot.Working_User.Uid != "0" {

			target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
			if !RWX_JUDGE(target_dir_inode, "w") {
				in_out.Out("当前用户权限不足，无法在该目录新建目录！")
				return
			}
		}

		isSuccess := Cd(file, args[:1], 0)
		if !isSuccess {
			in_out.Out("找不到目录！")
		}
		mk_dir(file, args[1])

		boot.Working_Directory = now_dir_back
		boot.Working_INode = now_dir_inode_back
		boot.Working_Dentry = now_dir_dentry_back

	}

}

func mk_dir(file *os.File, target_dir string) {

	var name_list []string
	inode_list := inode.Get_Dentry(file, boot.Working_Dentry).Content
	// 经典遍历
	for i := 0; i < len(inode_list); i++ {
		temp_inode := inode.Get_INode(file, inode_list[i])
		if temp_inode.Filetype == "dir" {
			name_list = append(name_list, temp_inode.Filename)
		}
	}

	for i := 0; i < len(name_list); i++ {
		if name_list[i] == target_dir {
			in_out.Out("已有同名目录！")
			return
		}
	}

	temp_IMap := imap.IMap
	temp_IMap_Block := imap.IB
	temp_ZMap := zmap.ZMap
	temp_ZMap_Block := zmap.ZB

	temp_dir_inode_index := imap.Get_Free_INode()
	temp_dir_dentry_index := zmap.Get_Free_Block()
	// 如果二者有一个申请失败，说明磁盘满了，要放弃创建目录
	if temp_dir_dentry_index == -1 || temp_dir_dentry_index == -1 {
		// 开始恢复现场，用刚才暂存的数据重新写入内存（这时候磁盘还没来得及写入，不用管）
		in_out.Out("disk磁盘空间不足，新建目录失败！")
		imap.IMap = temp_IMap
		imap.Free_INode = temp_IMap_Block.Free_INode
		imap.Last_position = temp_IMap_Block.Last_position

		zmap.ZMap = temp_ZMap
		zmap.Free_Block = temp_ZMap_Block.Free_Block
		zmap.Last_position = temp_ZMap_Block.Last_position
		zmap.Disk_size_remain = temp_ZMap_Block.Disk_size_remain

		return
	}

	var temp_control [3]int8
	// root用户建目录默认755，其他用户775
	if boot.Working_User.Uid == "0" {
		temp_control = [3]int8{7, 5, 5}
	} else {
		temp_control = [3]int8{7, 7, 5}
	}

	// 填充inode
	curr_time := time.Now()
	temp_dir_inode := inode.INode{
		Filename:    target_dir,
		Filetype:    "dir",
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     temp_control,
		Owner_id:    boot.Working_User.Uid,
		Group_id:    boot.Working_User.Gid,
		Zone:        [10]int{temp_dir_dentry_index},
	}

	// 获取全路径名
	full_dname := boot.Working_Directory
	if boot.Working_Directory == "/" {
		full_dname += target_dir
	} else {
		full_dname += ("/" + target_dir)
	}
	// 填充dentry
	temp_dir_dentry := inode.Dentry{
		To_INode:    temp_dir_inode_index,
		To_Father:   boot.Working_INode,
		DName:       full_dname,
		DName_Short: target_dir,
		Content:     make([]int, 0),
	}

	// 开始写入

	inode.Write_INode(file, temp_dir_inode, temp_dir_inode_index)
	inode.Write_Dentry(file, temp_dir_dentry, temp_dir_dentry_index)

	imap.Write_IMap(file)
	imap.IB.Free_INode = imap.Free_INode
	imap.IB.Last_position = imap.Last_position
	imap.Write_IMap_Block(file, imap.IB)

	zmap.Write_ZMap(file)
	zmap.ZB.Free_Block = zmap.Free_Block
	zmap.ZB.Last_position = zmap.Last_position
	zmap.ZB.Disk_size_remain = zmap.Disk_size_remain
	zmap.Write_ZMap_Block(file, zmap.ZB)

	// 向所在目录写入新目录信息
	temp_working_dir_dentry := inode.Get_Dentry(file, boot.Working_Dentry)
	temp_working_dir_dentry.Content = append(temp_working_dir_dentry.Content, temp_dir_inode_index)

	inode.Write_Dentry(file, temp_working_dir_dentry, boot.Working_Dentry)

	// 如果所在目录是根目录，则要及时更新这些
	if temp_working_dir_dentry.To_INode == 0 {
		inode.Root_INode = inode.Get_INode(file, 0)
		inode.Root_Dentry = inode.Get_Dentry(file, 0)
	}
}
