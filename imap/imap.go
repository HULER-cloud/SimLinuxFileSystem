package imap

import (
	"encoding/json"
	"golangsfs/basic"
	"golangsfs/boot"
	"os"
)

const IMAP_END = (boot.ZONE_START-boot.INODE_START)/8 - 1

const IMAP_BLOCK_POS = 3200

type IMap_Block struct {
	Free_INode    int
	Last_position int
}

var IB IMap_Block

var IMap []byte    // 内存中的inode位图
var Free_INode int // 可用inode数量

var Last_position = 0 // 上次找到的空闲inode对应位所在byte的位置
// 也就是上次修改过的byte位置

func Get_IMap(file *os.File) {
	// 读出4个块
	IMap = make([]byte, 4)
	file.Seek(boot.IMAP_START*1024, 0)
	file.Read(IMap)
}

func Write_IMap(file *os.File) {
	// 写入IMap，4个块
	file.Seek(boot.IMAP_START*1024, 0)
	file.Write(IMap)
}

func Get_Free_INode() (index int) {
	// 从上次修改过的byte块开始找
	for i := Last_position; i <= IMAP_END; i++ {
		for j := 0; j < 8; j++ {
			// 表明找到某一位为0，即inode为空
			if IMap[i]&byte(1<<(7-j)) == 0 {
				// 设置上次修改过的byte位置
				Last_position = i
				// 将找到的空闲位改为1
				IMap[i] = IMap[i] | byte(1<<(7-j))
				// 空闲inode数减1
				Free_INode--
				return i*8 + j
			}
		}
	}
	// 如果找到末尾还没有找到，就从头开始找前半段
	for i := 0; i < Last_position; i++ {
		for j := 0; j < 8; j++ {
			if IMap[i]&byte(1<<(7-j)) == 0 {
				Last_position = i
				IMap[i] = IMap[i] | byte(1<<(7-j))
				Free_INode--
				return i*8 + j
			}
		}
	}
	// 如果都没找到那就是inode块已经穷尽了

	return -1
}

func Get_IMap_Block(file *os.File) IMap_Block {
	// 将文件指针移动到IMap的位置
	file.Seek(boot.IMAP_START*1024+IMAP_BLOCK_POS, 0)

	// 创建IMap_Block内存指针
	ib := &IMap_Block{}
	// 创建接收长度的byte数组
	struct_length := make([]byte, 4)
	// 读出结构体json长度
	file.Read(struct_length)
	// 根据上面的长度继续读出整个结构体的数据
	ib_byte := make([]byte, basic.BytesToInt(struct_length))
	file.Read(ib_byte)
	// 反序列化成为真正的结构体
	json.Unmarshal(ib_byte, ib)

	return *ib
}

func Write_IMap_Block(file *os.File, ib IMap_Block) {
	// 将文件指针移动到IMap_Block的位置
	file.Seek(boot.IMAP_START*1024+IMAP_BLOCK_POS, 0)
	// 将IMap_Block结构体序列化为json字符串，用以存储
	ib_json, _ := json.Marshal(ib)
	// 写入长度信息
	file.Write(basic.IntToBytes(len(ib_json)))
	// 写入结构体信息
	file.Write(ib_json)

}
