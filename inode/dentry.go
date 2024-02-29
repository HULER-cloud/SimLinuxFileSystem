package inode

import (
	"encoding/json"
	"golangsfs/basic"
	"golangsfs/boot"
	"os"
)

// 目录项对象结构体，是加载到内存中有的文件逻辑结构，在simdisk中不存在物理存储
// 注意，目录项和目录的概念不一样，目录在simdisk中有其物理存储，也有对应的inode

// Dentry目前还没有完全确定，后面还会改，目前先这样

type Dentry struct {

	// 各种关联结构
	To_INode  int // 指向同一文件的inode
	To_Father int // 指向父级目录项的inode
	// 目录项信息
	DName       string // 目录项名称
	DName_Short string // 短名称

	// 目录下信息，是目录中所含成员的inode列表
	Content []int
}

var Root_Dentry Dentry // 根目录的dentry

func Get_Dentry(file *os.File, pos int) Dentry {
	// 将文件指针移动到目标dentry的block的位置
	file.Seek(int64((boot.ZONE_START+pos)*1024), 0)
	// 创建dentry文件块内存指针
	de := &Dentry{}
	// 创建接收长度的byte数组
	struct_length := make([]byte, 4)
	// 读出结构体json长度
	file.Read(struct_length)
	// 根据上面的长度继续读出整个结构体
	dentry_byte := make([]byte, basic.BytesToInt(struct_length))
	file.Read(dentry_byte)
	// 反序列化成为真正的结构体
	json.Unmarshal(dentry_byte, de)

	return *de
}

func Write_Dentry(file *os.File, de Dentry, pos int) {
	// 将文件指针移动到目标inode的位置
	file.Seek(int64((boot.ZONE_START+pos)*1024), 0)
	// 将dentry文件结构体序列化为json字符串，用以存储
	dentry_json, _ := json.Marshal(de)
	// 写入长度信息
	file.Write(basic.IntToBytes(len(dentry_json)))
	// 写入结构体信息
	file.Write(dentry_json)
}
