package zmap

import (
	"encoding/json"
	"golangsfs/basic"
	"golangsfs/boot"
	"os"
)

const ZMAP_END = (boot.DISK_SIZE_BLOCK-boot.ZONE_START)/8 - 1

const ZMAP_BLOCK_POS = 9600

// 结构体仅写入用
type ZMap_Block struct {
	Free_Block       int
	Disk_size_remain int
	Last_position    int
}

var ZB ZMap_Block

var ZMap []byte
var Free_Block int // 可用文件块数量
var Disk_size_remain int

var Last_position = 0 // 上次找到的空闲文件块对应位所在byte的位置
// 也就是上次修改过的byte位置

func Get_ZMap(file *os.File) {
	// 读出10个块
	ZMap = make([]byte, 10)
	file.Seek(boot.ZMAP_START*1024, 0)
	file.Read(ZMap)
}

func Write_ZMap(file *os.File) {
	// 写入ZMap，10个块
	file.Seek(boot.ZMAP_START*1024, 0)
	file.Write(ZMap)
}

func Get_Free_Block() (index int) {
	// 从上次修改过的byte块开始找
	for i := Last_position; i <= ZMAP_END; i++ {
		for j := 0; j < 8; j++ {
			// 表明找到某一位为0，即文件块为空
			if ZMap[i]&uint8(1<<(7-j)) == 0 {
				// 设置上次修改过的byte位置
				Last_position = i
				// 将找到的空闲位改为1
				ZMap[i] = ZMap[i] | uint8(1<<(7-j))
				// 空闲文件块数减1
				Free_Block--
				// 磁盘剩余空间减1
				Disk_size_remain--
				return i*8 + j
			}
		}
	}
	// 如果找到末尾还没有找到，就从头开始找前半段
	for i := 0; i < Last_position; i++ {
		for j := 0; j < 8; j++ {
			if ZMap[i]&uint8(1<<(7-j)) == 0 {
				Last_position = i
				ZMap[i] = ZMap[i] | uint8(1<<(7-j))
				Free_Block--
				Disk_size_remain--
				return i*8 + j
			}
		}
	}
	// 如果都没找到那就是inode块已经穷尽了
	return -1
}

func Get_ZMap_Block(file *os.File) ZMap_Block {
	// 将文件指针移动到ZMap的位置
	file.Seek(boot.ZMAP_START*1024+ZMAP_BLOCK_POS, 0)
	// 创建ZMap_Block内存指针
	zb := &ZMap_Block{}
	// 创建接收长度的byte数组
	struct_length := make([]byte, 4)
	// 读出结构体json长度
	file.Read(struct_length)
	// 根据上面的长度继续读出整个结构体的数据
	zb_byte := make([]byte, basic.BytesToInt(struct_length))
	file.Read(zb_byte)
	// 反序列化成为真正的结构体
	json.Unmarshal(zb_byte, zb)

	return *zb
}

func Write_ZMap_Block(file *os.File, zb ZMap_Block) {
	// 将文件指针移动到ZMap_Block的位置
	file.Seek(boot.ZMAP_START*1024+ZMAP_BLOCK_POS, 0)
	// 将Zmap_Block结构体序列化为json字符串，用以存储
	zb_json, _ := json.Marshal(zb)
	// 写入长度信息
	file.Write(basic.IntToBytes(len(zb_json)))
	// 写入结构体信息
	file.Write(zb_json)

}
