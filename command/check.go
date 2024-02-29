package command

import (
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"golangsfs/super"
	"golangsfs/writing"
	"golangsfs/zmap"
	"os"
)

func Check(file *os.File) {
	// 防止功能执行中由于错误发生，工作目录相关数据来不及写回情况
	// 进行重置回根目录的操作
	boot.Working_Directory = "/"
	boot.Working_INode = 0
	boot.Working_Dentry = 0

	// 防止磁盘误写，一些关键数据结构从运行内存中读取出来并正确写入

	boot.Write_Boot_Block(file, boot.BB)

	super.Write_Super_Block(file, super.SB)

	imap.Write_IMap(file)
	imap.Write_IMap_Block(file, imap.IB)

	zmap.Write_ZMap(file)
	zmap.Write_ZMap_Block(file, zmap.ZB)

	inode.Write_INode(file, inode.Root_INode, 0)
	inode.Write_Dentry(file, inode.Root_Dentry, 0)

	// 写入标记恢复为0，因为已经到了要恢复的地步了，无人可以正常写
	writing.To_Not_Writing(file)

	in_out.Out("修复完成！")

}
