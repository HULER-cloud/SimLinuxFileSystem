package writing

import (
	"golangsfs/in_out"
	"os"
	"time"
)

func To_Writing(file *os.File) {
	// 找到标记位位置
	file.Seek(1200, 0)
	// 写入1
	mark := []byte{1}
	file.Write(mark)
}

func To_Not_Writing(file *os.File) {
	// 同理
	file.Seek(1200, 0)
	mark := []byte{0}
	file.Write(mark)
}

func Get_Mark(file *os.File) int {
	// 读出标记位
	file.Seek(1200, 0)
	mark := make([]byte, 1)
	file.Read(mark)
	return int(mark[0])
}

func Writing_Test(file *os.File) {
	if Get_Mark(file) == 1 {
		in_out.Out("磁盘被其他进程写入中，当前进程被阻塞！")
		for {
			// 睡眠的方式来定时查询标记位
			time.Sleep(1 * time.Second)
			if Get_Mark(file) == 0 {
				in_out.Out("其他进程写入完毕，从阻塞中恢复并完成任务！")
				break
			}
		}
	}
}
