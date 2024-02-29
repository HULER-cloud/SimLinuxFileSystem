package command

import (
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/in_out"
	"golangsfs/inode"
	"os"
)

func Cat(file *os.File, args []string) (data []byte) {
	// 文件名
	filename := args[0]
	// 获取当前工作目录的dentry
	now_dir_dentry := inode.Get_Dentry(file, boot.Working_Dentry)
	// 没有找到文件的标记flag
	notfound := true
	// 遍历查找目的文件
	for i := 0; i < len(now_dir_dentry.Content); i++ {
		temp_inode := inode.Get_INode(file, now_dir_dentry.Content[i])
		temp_filename := temp_inode.Filename
		// 拼凑完整文件名
		if temp_inode.Filetype != "" {
			temp_filename += ("." + temp_inode.Filetype)
		}

		// 找到了！
		if filename == temp_filename {
			notfound = false
			// 比较特殊，找到了才能进行权限控制
			if boot.Working_User.Uid != "0" {
				target_file_inode := temp_inode

				if !RWX_JUDGE(target_file_inode, "read") {
					in_out.Out("当前用户权限不足，无法查看该文件内容！")
					return
				}
			}
			// 返回字节流数组
			return read(file, temp_inode)
		}
	}
	// 没找到……
	if notfound {
		in_out.Out("目标文件不存在！")
		return make([]byte, 0)
	}
	return make([]byte, 0)
}

func read(file *os.File, file_inode inode.INode) (data []byte) {
	// 直接块
	if file_inode.File_size <= inode.Direct_size {
		for i := 0; i < 8; i++ {
			// 说明这个块有数据
			if file_inode.Zone[i] != 0 {
				temp_add_data := read_block_inside(file, file_inode.Zone[i])
				for j := 0; j < len(temp_add_data); j++ {
					// 字节数组追加数据
					data = append(data, temp_add_data[j])
				}
			}
		}
	} else if file_inode.File_size <= inode.First_size { // 一级间接块
		// 直接块可以全部读出
		for i := 0; i < 8; i++ {
			temp_add_data := read_block_inside(file, file_inode.Zone[i])
			for j := 0; j < len(temp_add_data); j++ {
				data = append(data, temp_add_data[j])
			}
		}

		// 开始读一次间接块
		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < 256; i++ {
			extra_block := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			// 说明这4个字节指向一个存数据的文件块
			if extra_block != 0 {
				temp_add_data := read_block_inside(file, extra_block)
				for j := 0; j < len(temp_add_data); j++ {
					data = append(data, temp_add_data[j])
				}
			} else {
				break
			}
		}

	} else { // 二级间接块
		// 直接块可以全部读出
		for i := 0; i < 8; i++ {
			temp_add_data := read_block_inside(file, file_inode.Zone[i])
			for j := 0; j < len(temp_add_data); j++ {
				data = append(data, temp_add_data[j])
			}
		}

		// 开始读一次间接块，也可以全部读出
		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < 256; i++ {
			extra_block := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			temp_add_data := read_block_inside(file, extra_block)
			for j := 0; j < len(temp_add_data); j++ {
				data = append(data, temp_add_data[j])
			}
		}

		// 开始读二次间接块
		first_block = inode.Get_Block(file, file_inode.Zone[9])
		for i := 0; i < 256; i++ {
			extra_block := basic.BytesToInt(first_block[i : i+1])
			breakFlag := false
			if extra_block != 0 {
				second_block := inode.Get_Block(file, extra_block)
				for j := 0; j < 256; j++ {
					extra_block_2 := basic.BytesToInt(second_block[j : j+1])
					// 说明这4个字节指向的文件块有数据
					if extra_block_2 != 0 {
						temp_add_data := read_block_inside(file, extra_block_2)
						for k := 0; k < len(temp_add_data); k++ {
							data = append(data, temp_add_data[k])
						}
					} else {
						breakFlag = true
						break
					}
				}
				if breakFlag {
					break
				}
			} else {
				break
			}
		}
	}
	return data
}

// 读文件块内部的数据
func read_block_inside(file *os.File, pos int) (data []byte) {
	// 先获取到块
	temp_block := inode.Get_Block(file, pos)

	// 因为一个块中数据可能不是全都有，后面可能都是0
	end := 0
	for i := 0; i < 1024; i++ {
		if temp_block[i] != 0 {
			end++
		}
	}
	// 只返回有数据的部分
	return temp_block[:end]
}
