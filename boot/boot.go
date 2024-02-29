package boot

import (
	"encoding/json"
	"golangsfs/basic"
	"os"
	"os/user"
)

// Boot_Block 启动引导块存放在磁盘的第0号块
// 并非磁盘的boot，而是simdisk程序的boot

type Boot_Block struct {
	Next_UID   int
	Next_GID   int
	User_list  map[int]user.User
	Group_list map[int]user.Group
}

const DISK_SIZE_BLOCK int = 100 * 1024

const DISK_SIZE_BYTE int = 100 * 1024 * 1024

const DISK_PATH_WHOLE string = "./simdisk"

const BOOT_START = 0

const SUPER_BLOCK_START = 1

const IMAP_START = 2 // IMap: 2~5, 共4块

const ZMAP_START = 6 // ZMap: 6~15, 共10块
// 为了凑8的整方便各种查找以及搜索数据，这里两个位图的块比就不弄成1：3了

// 下面两个需要解释一下，在Linux中inode块与实际文件块的比例大约是1：3
// 102400个块除去前面18个之外还剩102382个，按1：3的比例分配
// 结果是25595.5对76786.5
// 为了凑整+方便查找，我们让文件块的开始位置设置在25600，相对比例几乎不变

const INODE_START = 16 // inode: 16~25599, 共25584块
// 25582/8=3198，代表IMap需要3198B，我们预留出来的4块足矣

const ZONE_START = 25600 // zone: 25600~102399, 共76800块
// 76800/8=9600，代表ZMap需要9600B，我们预留出来的10块足矣
var BB Boot_Block

var Working_Directory string // 当前工作目录

var Working_INode int // 当前工作inode编号

var Working_Dentry int // 当前工作dentry编号

var Last_UID int // 下一个分配的UID

var Last_GID int // 下一个分配的GID

var Working_User user.User // 当前工作用户

var Root_user = user.User{
	Uid:      "0",
	Gid:      "0",
	Username: "root",
	Name:     "root",
	HomeDir:  "/root",
}

var Root_group = user.Group{
	Gid:  "0",
	Name: "rootgroup",
}

var Ordinary_user_default = user.User{
	Uid:      "1",
	Gid:      "1",
	Username: "default",
	Name:     "default",
	HomeDir:  "/home/default",
}

var Oridinary_group = user.Group{
	Gid:  "1",
	Name: "ordinary",
}

var Ordinary_user_second = user.User{
	Uid:      "2",
	Gid:      "1",
	Username: "second",
	Name:     "second",
	HomeDir:  "/home/second",
}

var Ordinary_user_outside = user.User{
	Uid:      "3",
	Gid:      "2",
	Username: "another",
	Name:     "another",
	HomeDir:  "/home/another",
}

var Another_group = user.Group{
	Gid:  "2",
	Name: "another",
}

var User_list = make(map[int]user.User)

var Group_list = make(map[int]user.Group)

func Get_Boot_Block(file *os.File) Boot_Block {
	// 将文件指针移动到boot块的位置
	file.Seek(BOOT_START, 0)
	// 创建boot块内存指针
	bb := &Boot_Block{}
	// 创建接收长度的byte数组
	struct_length := make([]byte, 4)
	// 读出结构体json长度
	file.Read(struct_length)
	// 根据上面的长度继续读出整个结构体的数据
	bb_byte := make([]byte, basic.BytesToInt(struct_length))
	file.Read(bb_byte)
	// 反序列化成为真正的结构体
	json.Unmarshal(bb_byte, bb)

	return *bb
}

func Write_Boot_Block(file *os.File, bb Boot_Block) {
	// 将文件指针移动到boot块的位置
	file.Seek(BOOT_START, 0)
	// 将boot块结构体序列化为json字符串，用以存储
	bb_json, _ := json.Marshal(bb)
	// 写入长度信息
	file.Write(basic.IntToBytes(len(bb_json)))
	// 写入结构体信息
	file.Write(bb_json)
}
