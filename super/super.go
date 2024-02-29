package super

import (
	"encoding/json"
	"golangsfs/basic"
	"golangsfs/boot"
	"os"
)

// Super_Block 超级块存放在磁盘的第1号块
type Super_Block struct {
	Max_file_size int // 单个文件最大大小

	IMap_blocks int // inode位图占用块数

	ZMap_blocks int // 逻辑块数位图占用块数

	INodes int // inode数

	Blocks int // 总共的文件块数

}

var SB Super_Block

const MAX_FILE_SIZE = 65800

func Get_Super_Block(file *os.File) Super_Block {
	// 将文件指针移动到超级块的位置
	file.Seek(boot.SUPER_BLOCK_START*1024, 0)
	// 创建超级块内存指针
	sb := &Super_Block{}
	// 创建接收长度的byte数组
	struct_length := make([]byte, 4)
	// 读出结构体json长度
	file.Read(struct_length)
	// 根据上面的长度继续读出整个结构体的数据
	sb_byte := make([]byte, basic.BytesToInt(struct_length))
	file.Read(sb_byte)
	// 反序列化成为真正的结构体
	json.Unmarshal(sb_byte, sb)

	return *sb
}

func Write_Super_Block(file *os.File, sb Super_Block) {
	// 将文件指针移动到超级块的位置
	file.Seek(boot.SUPER_BLOCK_START*1024, 0)
	// 将超级块结构体序列化为json字符串，用以存储
	sb_json, _ := json.Marshal(sb)
	// 写入长度信息
	file.Write(basic.IntToBytes(len(sb_json)))
	// 写入结构体信息
	file.Write(sb_json)

}
