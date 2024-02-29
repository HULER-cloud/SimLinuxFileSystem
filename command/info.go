package command

import (
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/super"
	"golangsfs/zmap"
	"strconv"
)

func Info() {

	in_out.Out("simdisk模拟磁盘文件管理系统信息如下：")
	in_out.Out("")
	in_out.Out("系统用户信息：")
	in_out.Out("当前用户：" + boot.Working_User.Username)
	in_out.Out("用户uid" + boot.Working_User.Uid)
	in_out.Out("所属组id：" + boot.Working_User.Gid)
	in_out.Out("用户列表：")
	in_out.Out("用户uid " + "用户名 " + "用户组id")
	for i := 0; i < len(boot.User_list); i++ {
		in_out.Out(boot.User_list[i].Uid + " " + boot.User_list[i].Username +
			" " + boot.User_list[i].Gid)
	}

	in_out.Out("")

	in_out.Out("波动属性：")
	in_out.Out("已用磁盘容量 " + strconv.Itoa(super.SB.Blocks-zmap.Disk_size_remain/1024) + " KB")
	in_out.Out("剩余磁盘容量 " + strconv.Itoa(zmap.Disk_size_remain/1024) + " KB")
	in_out.Out("已用inode数 " + strconv.Itoa(super.SB.INodes-imap.Free_INode))
	in_out.Out("可用inode数 " + strconv.Itoa(imap.Free_INode))

	in_out.Out("")

	in_out.Out("固有属性：")
	in_out.Out("磁盘总容量：" + strconv.Itoa(super.SB.Blocks*1024) + "KB")
	in_out.Out("最大文件大小：" + strconv.Itoa(super.SB.Max_file_size) + "KB")
	in_out.Out("inode位图区块大小：" + strconv.Itoa(super.SB.IMap_blocks))
	in_out.Out("文件块位图区块大小：" + strconv.Itoa(super.SB.ZMap_blocks))
	in_out.Out("inode最大数量：" + strconv.Itoa(super.SB.INodes))
	in_out.Out("总文件块数：" + strconv.Itoa(super.SB.Blocks))

}
