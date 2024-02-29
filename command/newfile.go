package command

import (
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"os"
	"time"
)

func Newfile(file *os.File, args []string) (newfile_inode inode.INode, newfile_inode_index int) {
	// 权限控制
	if boot.Working_User.Uid != "0" {
		target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
		if !RWX_JUDGE(target_dir_inode, "update") {
			in_out.Out("当前用户权限不足，无法在该目录中新建文件！")
			return
		}
	}

	_, target_filename, target_filetype := basic.Analyse_file_path("./" + args[0])
	_, _, ok := Is_target_file_exist(file, inode.Get_Dentry(file, boot.Working_Dentry), target_filename, target_filetype)
	if ok == true {
		in_out.Out("当前目录已存在同名文件！")
		return
	}

	_, filename, filetype := basic.Analyse_file_path(args[0])

	curr_time := time.Now()
	var temp_control [3]int8
	// root用户建文件默认644，其他用户664
	if boot.Working_User.Uid == "0" {
		temp_control = [3]int8{6, 4, 4}
	} else {
		temp_control = [3]int8{6, 6, 4}
	}

	// 填充inode
	newfile_inode = inode.INode{
		Filename:    filename,
		Filetype:    filetype,
		File_size:   0,
		Create_time: curr_time,
		Modify_time: curr_time,
		Control:     temp_control,
		Owner_id:    boot.Working_User.Uid,
		Group_id:    boot.Working_User.Gid,
		Zone:        [10]int{},
	}

	newfile_inode_index = imap.Get_Free_INode()

	if newfile_inode_index == -1 {
		in_out.Out("磁盘inode耗尽，新建文件失败！")
		return inode.INode{}, -1
	}

	inode.Write_INode(file, newfile_inode, newfile_inode_index)

	imap.Write_IMap(file)
	imap.IB.Free_INode = imap.Free_INode
	imap.IB.Last_position = imap.Last_position
	imap.Write_IMap_Block(file, imap.IB)

	// 向上级目录写入信息

	temp_working_dir_dentry := inode.Get_Dentry(file, boot.Working_Dentry)
	temp_working_dir_dentry.Content = append(temp_working_dir_dentry.Content, newfile_inode_index)

	inode.Write_Dentry(file, temp_working_dir_dentry, boot.Working_Dentry)

	// 更新内存中root相关
	if boot.Working_Directory == "/" {
		inode.Root_INode = inode.Get_INode(file, 0)
		inode.Root_Dentry = inode.Get_Dentry(file, 0)
	}

	// 任务三测试使用
	// 所以说，不建议使用second用户登录，要不然就很容易出现奇怪情况
	if boot.Working_User.Username == "second" {
		time.Sleep(30 * time.Second)
	}

	return newfile_inode, newfile_inode_index
}
