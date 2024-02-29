package basic

import (
	"bytes"
	"encoding/binary"
	"os"
)

// 清除某多个块（[start, end]）全部数据的函数，
func Clean(file *os.File, start int, end int) {
	// 移动到start块位置
	file.Seek(int64(start*1024), 0)
	// 生成(end-start+1)*1024大小的空字节数组
	zeros := make([]byte, (end-start+1)*1024)
	// 覆盖写入到磁盘文件中
	file.Write(zeros)
}

// 整形转换成字节，小端模式
func IntToBytes(n int) []byte {
	// 化为4字节int32
	tmp := int32(n)
	// 创建字节数组
	bytesBuffer := bytes.NewBuffer([]byte{})
	// 小端模式写入
	binary.Write(bytesBuffer, binary.LittleEndian, tmp)
	return bytesBuffer.Bytes()
}

// 字节转换成整形，小端模式
func BytesToInt(b []byte) int {
	// 复制一个字节数组备份
	bytesBuffer := bytes.NewBuffer(b)
	var tmp int32
	// 小端模式读出到int32数据中
	binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
	return int(tmp)
}

// 分析路径，传入一个完整路径，返回分开的目录、文件名、文件类型
func Analyse_file_path(fullpath string) (dir string, filename string, filetype string) {
	// 初始化点号和斜杠标记位
	dot_pos := -1
	last_slash_pos := -1
	// 更新点号标记位
	for i := len(fullpath) - 1; i >= 0; i-- {
		if fullpath[i] == '.' {
			dot_pos = i
			break
		}
	}
	// 更新斜杠标记位
	for i := len(fullpath) - 1; i >= 0; i-- {
		if fullpath[i] == '/' || fullpath[i] == '\\' {
			last_slash_pos = i
			break
		}
	}
	// 都没变动，传入的就是纯文件名，且无后缀名
	if dot_pos == -1 && last_slash_pos == -1 {
		dir = "./"
		filename = fullpath
		filetype = ""
	} else if dot_pos == -1 { // 传入为目录+文件，但无后缀名
		dir = fullpath[:last_slash_pos]
		filename = fullpath[last_slash_pos+1:]
		filetype = ""
	} else if last_slash_pos == -1 { // 传入为带后缀名的文件名
		dir = "./"
		filename = fullpath[:dot_pos]
		filetype = fullpath[dot_pos+1:]

	} else { // 都有，分开来就行
		dir = fullpath[:last_slash_pos]
		// 防止由于 / 前面没有东西而获取不到目录，这样情况就是根目录
		if last_slash_pos == 0 {
			dir = "/"
		}
		filename = fullpath[last_slash_pos+1 : dot_pos]
		filetype = fullpath[dot_pos+1:]
	}

	return dir, filename, filetype

}
