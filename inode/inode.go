package inode

import (
	"encoding/json"
	"golangsfs/basic"
	"golangsfs/boot"
	"os"
	"time"
)

// 结构体声明这里实际上也可以不去限制固定长度存储，转存到文件里面的时候手动指定和读取就可以
// 可以暂时不必纠结

// 一个额外的知识点，linux中inode块和文件块的数量比大致是1:3，绝大部分块可以按照这样的数量直接划分了

type INode struct {

	// 文件信息相关
	Filename    string
	Filetype    string    // 还不一定能够直接是string，得调整，暂时占位
	File_size   int       // 以字节为单位的文件大小
	Create_time time.Time // 创建时间，时间暂时都用UTC时间，后面有需要再改
	Modify_time time.Time // 最后修改时间

	// 控制权限相关
	Control  [3]int8 // linux经典控制权限，文件默认权限是644，目录默认权限是755
	Owner_id string  // 文件拥有者user.User标识, user.User.Uid
	Group_id string  // 拥有者所在组user.Group标识, user.Group.Gid，组一般会少一些，ID号也会小一些

	// 文件内容相关

	// 0~7为直接块，8为一级间接块，9为二级间接块
	Zone [10]int
	// 算一下各自对应的大小
	// 0~7一共8个块，0<=filesize<=8KB
	// 8+256=264个块，8KB<filesize<=264KB
	// 8+256+256*256=65800个块,264KB<filesize<=65800KB
	// 到达了disk大小的一多半（实际上如果只看可用容量已经差不多了）所以停止

}

var Root_INode INode

const Direct_size = 8

const First_size = 264

const Second_size = 65800

func Get_INode(file *os.File, pos int) INode {
	// 将文件指针移动到目标inode的位置
	file.Seek(int64((boot.INODE_START+pos)*1024), 0)
	// 创建inode块内存指针
	inode := &INode{}
	// 创建接收长度的byte数组
	struct_length := make([]byte, 4)
	// 读出结构体json长度
	file.Read(struct_length)
	// 根据上面的长度继续读出整个结构体
	inode_byte := make([]byte, basic.BytesToInt(struct_length))
	file.Read(inode_byte)
	// 反序列化成为真正的结构体
	json.Unmarshal(inode_byte, inode)

	return *inode
}

func Write_INode(file *os.File, inode INode, pos int) {
	// 将文件指针移动到目标inode的位置
	file.Seek(int64((boot.INODE_START+pos)*1024), 0)
	// 将INode结构体序列化为json字符串，用以存储
	inode_json, _ := json.Marshal(inode)
	// 写入长度信息
	file.Write(basic.IntToBytes(len(inode_json)))
	// 写入结构体信息
	file.Write(inode_json)
}
