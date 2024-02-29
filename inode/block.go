package inode

import (
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/zmap"
	"os"
)

func Get_Block(file *os.File, pos int) (data []byte) {
	// 将文件指针移动到目标文件块的位置
	file.Seek(int64((boot.ZONE_START+pos)*1024), 0)
	data = make([]byte, 1024)
	file.Read(data)
	return data
}

func Write_Block(file *os.File, data []byte, pos int) {
	// 将文件指针移动到目标文件块的位置
	file.Seek(int64((boot.ZONE_START+pos)*1024), 0)
	file.Write(data)
}

func Free_Certain_Block(file *os.File, pos int) {
	// 首先清掉块本身
	basic.Clean(file, boot.ZONE_START+pos, boot.ZONE_START+pos)
	// 再修改zmap
	// 获取到block对应的zmap的字节位和比特位
	byte_index := pos / 8
	bit_index := pos % 8
	// 修改、写回
	zmap.ZMap[byte_index] = ^(zmap.ZMap[byte_index] & (1 << (7 - bit_index)))
	zmap.Write_ZMap(file)
	zmap.Free_Block++
	zmap.Disk_size_remain++
	zmap.ZB.Free_Block = zmap.Free_Block
	zmap.ZB.Disk_size_remain = zmap.Disk_size_remain
	zmap.Write_ZMap_Block(file, zmap.ZB)

}
