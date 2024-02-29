package command

import (
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"golangsfs/zmap"
	"os"
	"time"
)

func Is_target_dir_exist(file *os.File, path string) (dir_inode inode.INode,
	dir_dentry inode.Dentry, isSuccess bool) {
	now_dir_back := boot.Working_Directory
	now_dir_inode_back := boot.Working_INode
	now_dir_dentry_back := boot.Working_Dentry

	// 尝试切到目标目录
	isSuccess = Cd(file, []string{path}, 0)
	if !isSuccess {
		// 切不过去说明不存在
		boot.Working_Directory = now_dir_back
		boot.Working_INode = now_dir_inode_back
		boot.Working_Dentry = now_dir_dentry_back
		return inode.INode{}, inode.Dentry{}, false
	}
	dir_inode = inode.Get_INode(file, boot.Working_INode)
	dir_dentry = inode.Get_Dentry(file, boot.Working_Dentry)

	boot.Working_Directory = now_dir_back
	boot.Working_INode = now_dir_inode_back
	boot.Working_Dentry = now_dir_dentry_back
	return dir_inode, dir_dentry, true
}

func Is_target_file_exist(file *os.File, now_dir_dentry inode.Dentry,
	filename string, filetype string) (dir_inode inode.INode,
	file_inode_index int, isSuccess bool) {
	// 遍历查找即可
	for i := 0; i < len(now_dir_dentry.Content); i++ {
		temp_inode := inode.Get_INode(file, now_dir_dentry.Content[i])
		if temp_inode.Filename == filename && temp_inode.Filetype == filetype {
			return temp_inode, now_dir_dentry.Content[i], true
		}
	}

	return inode.INode{}, -1, false

}

// 清空文件的zone，但是不删除inode，用于覆写
// 实际上也只实现了覆写……
func Del_zone(file *os.File, file_inode inode.INode, file_inode_index int) {
	if file_inode.Zone[0] == 0 {
		// 说明本来zone就是空的，没必要再清空
		return
	}
	// 获取当前时间作为修改时间
	curr_time := time.Now()
	file_inode.Modify_time = curr_time
	// 记录要清除的文件块
	var to_be_cleaned []int
	// 直接块
	for i := 0; i < 8; i++ {
		if file_inode.Zone[i] != 0 {
			to_be_cleaned = append(to_be_cleaned, file_inode.Zone[i])
		} else {
			break
		}

	}

	// 一级间接块
	if file_inode.Zone[8] != 0 {
		to_be_cleaned = append(to_be_cleaned, file_inode.Zone[8])
		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < 256; i++ {
			first_block_index := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			// 说明这4个字节指向的一级块有数据
			if first_block_index != 0 {
				to_be_cleaned = append(to_be_cleaned, first_block_index)
			} else {
				break
			}
		}
	}
	// 二级间接块
	if file_inode.Zone[9] != 0 {
		to_be_cleaned = append(to_be_cleaned, file_inode.Zone[9])
		second_block := inode.Get_Block(file, file_inode.Zone[9])
		for i := 0; i < 256; i++ {
			breakFlag := false
			second_block_index := basic.BytesToInt(second_block[i*4 : (i+1)*4])
			// 经典有双层……
			if second_block_index != 0 {
				to_be_cleaned = append(to_be_cleaned, second_block_index)
				second_block_inside := inode.Get_Block(file, second_block_index)
				for j := 0; j < 256; j++ {
					second_block_inside_index := basic.BytesToInt(second_block_inside[j*4 : (j + 1*4)])
					// ……不等于0的if判断
					if second_block_inside_index != 0 {
						to_be_cleaned = append(to_be_cleaned, second_block_inside_index)
					} else {
						breakFlag = true
						break
					}
				}
			}
			if breakFlag {
				break
			}
		}
	}

	// 清空数据块并更新zmap相关
	for i := 0; i < len(to_be_cleaned); i++ {
		basic.Clean(file, boot.ZONE_START+to_be_cleaned[i], boot.ZONE_START+to_be_cleaned[i])
		byte_index := to_be_cleaned[i] / 8
		bit_index := to_be_cleaned[i] % 8
		zmap.ZMap[byte_index] = ^(zmap.ZMap[byte_index] & (1 << (7 - bit_index)))
		zmap.Free_Block++
		zmap.Disk_size_remain++
	}
	zmap.Write_ZMap(file)
	zmap.ZB.Free_Block = zmap.Free_Block
	zmap.Disk_size_remain = zmap.Disk_size_remain
	zmap.Write_ZMap_Block(file, zmap.ZB)

	// 最后清空zone
	file_inode.Zone = [10]int{}

	// inode写回到disk中
	inode.Write_INode(file, file_inode, file_inode_index)

}

func Allocate_zone(file *os.File, file_inode inode.INode, file_inode_index int, file_size int) {
	// 获取当前时间作为修改时间
	curr_time := time.Now()
	file_inode.Modify_time = curr_time

	// 文件大小是以字节为单位的
	// 需要占用的文件块数，也就是KB数（取整）
	block_nums := file_size/1024 + 1

	// 首先获取到所有需要的文件块index
	var all_zone_index []int

	// 需要额外获取的一级和二级间接块的index
	var first_block_index int

	// 除了二级间接块直接对应的block之外，还需要多少个block
	how_many := (block_nums-inode.First_size)/256 + 1

	second_block_index := make([]int, 1+how_many)
	if block_nums > inode.Direct_size {
		first_block_index = zmap.Get_Free_Block()
		if first_block_index == -1 {
			in_out.Out("磁盘空间不足！")
			free_all_zone_index(file, file_inode, file_inode_index, all_zone_index)
			return
		} else {
			all_zone_index = append(all_zone_index, first_block_index)
		}
	}
	if block_nums > inode.First_size {
		second_block_index[0] = zmap.Get_Free_Block()
		if second_block_index[0] == -1 {
			in_out.Out("磁盘空间不足！")
			free_all_zone_index(file, file_inode, file_inode_index, all_zone_index)
			return
		} else {
			all_zone_index = append(all_zone_index, second_block_index[0])
		}
		for i := 1; i < len(second_block_index); i++ {
			second_block_index[i] = zmap.Get_Free_Block()
			if second_block_index[i] == -1 {
				in_out.Out("磁盘空间不足！")
				free_all_zone_index(file, file_inode, file_inode_index, all_zone_index)
				return
			}
		}
	}

	// 运行到这里说明额外块已经申请完了，该申请数据文件块了
	// 申请文件块
	for i := 0; i < block_nums; i++ {
		temp_zone_index := zmap.Get_Free_Block()
		if temp_zone_index != -1 {
			all_zone_index = append(all_zone_index, temp_zone_index)
		} else {
			in_out.Out("磁盘空间不足！")
			// 运行到这里说明块获取不完全，需要释放掉已经获取的块
			free_all_zone_index(file, file_inode, file_inode_index, all_zone_index)
			return
		}

	}

	// 开始分配
	file_inode.File_size = len(all_zone_index)
	inside_index := 0
	if block_nums <= inode.Direct_size {
		// 一级不再赘述
		for i := 0; i < block_nums; i++ {
			file_inode.Zone[i] = all_zone_index[inside_index]
			inside_index++
		}

	} else if block_nums <= inode.First_size {
		inside_index = 1
		for i := 0; i < 8; i++ {
			file_inode.Zone[i] = all_zone_index[inside_index]
			inside_index++
		}
		// 写入一级间接块block
		file_inode.Zone[8] = all_zone_index[0]
		data := make([]byte, 1024)
		// 4个字节一次更新，直到一级间接块遍历完
		for i := 0; i < 256 && inside_index < len(all_zone_index); i++ {
			data[i*4] = basic.IntToBytes(all_zone_index[inside_index])[0]
			data[i*4+1] = basic.IntToBytes(all_zone_index[inside_index])[1]
			data[i*4+2] = basic.IntToBytes(all_zone_index[inside_index])[2]
			data[i*4+3] = basic.IntToBytes(all_zone_index[inside_index])[3]
			inside_index++
		}
		inode.Write_Block(file, data, all_zone_index[0])

	} else { // 外面调用申请的时候已经卡过file_size了，所以不用担心超上限
		inside_index = 1 + len(second_block_index)
		for i := 0; i < 8; i++ {
			file_inode.Zone[i] = all_zone_index[inside_index]
			inside_index++
		}

		// 写入一级间接块block
		file_inode.Zone[8] = all_zone_index[0]
		data := make([]byte, 1024)
		// 4个字节一次更新，直到一级间接块遍历完
		for i := 0; i < 256; i++ {
			data[i*4] = basic.IntToBytes(all_zone_index[inside_index])[0]
			data[i*4+1] = basic.IntToBytes(all_zone_index[inside_index])[1]
			data[i*4+2] = basic.IntToBytes(all_zone_index[inside_index])[2]
			data[i*4+3] = basic.IntToBytes(all_zone_index[inside_index])[3]
			inside_index++
		}

		inode.Write_Block(file, data, all_zone_index[0])

		// 写入二级间接块block
		file_inode.Zone[9] = all_zone_index[1]
		data = make([]byte, 1024)
		// 遍历外层的二级间接块block
		var temp_second_block_index []int
		for i := 2; i < 2+len(second_block_index); i++ {
			temp_second_block_index = append(temp_second_block_index, all_zone_index[inside_index])
			data[(i-2)*4] = basic.IntToBytes(all_zone_index[inside_index])[0]
			data[(i-2)*4+1] = basic.IntToBytes(all_zone_index[inside_index])[1]
			data[(i-2)*4+2] = basic.IntToBytes(all_zone_index[inside_index])[2]
			data[(i-2)*4+3] = basic.IntToBytes(all_zone_index[inside_index])[3]
			inside_index++
		}

		inode.Write_Block(file, data, all_zone_index[1])

		// 遍历内层的数据块block
		for i := 0; i < len(temp_second_block_index); i++ {
			data_2 := make([]byte, 1024)
			temp := inside_index
			for j := temp; j < len(all_zone_index); j++ {
				data_2[(j-temp)*4] = basic.IntToBytes(all_zone_index[inside_index])[0]
				data_2[(j-temp)*4+1] = basic.IntToBytes(all_zone_index[inside_index])[1]
				data_2[(j-temp)*4+2] = basic.IntToBytes(all_zone_index[inside_index])[2]
				data_2[(j-temp)*4+3] = basic.IntToBytes(all_zone_index[inside_index])[3]
				inside_index++
			}
			// 写入每个块

			inode.Write_Block(file, data_2, all_zone_index[i])

		}

	}

	zmap.Write_ZMap(file)
	zmap.ZB.Free_Block = zmap.Free_Block
	zmap.ZB.Disk_size_remain = zmap.Disk_size_remain
	zmap.ZB.Last_position = zmap.Last_position
	zmap.Write_ZMap_Block(file, zmap.ZB)

	inode.Write_INode(file, file_inode, file_inode_index)

}

// 在空间申请失败时，负责释放掉所有已经申请的块
func free_all_zone_index(file *os.File, file_inode inode.INode, file_inode_index int,
	all_zone_index []int) {
	for j := 0; j < len(all_zone_index); j++ {
		inode.Free_Certain_Block(file, all_zone_index[j])
	}

	// 最后清空zone
	file_inode.Zone = [10]int{}
	// 写回到disk中
	inode.Write_INode(file, file_inode, file_inode_index)

}

// 进行write前确保已经申请好空间
// 整体逻辑和Read_data一样，只是将读出的过程换成存入的过程
func Write_data(file *os.File, file_inode inode.INode, data []byte) {

	whole := file_inode.File_size

	// 凑整块字节，方便读取，不影响filesize
	for len(data)%1024 != 0 {
		data = append(data, 0)
	}

	if file_inode.File_size <= inode.Direct_size*1024 {
		for i := 0; i < whole; i++ {
			inode.Write_Block(file, data[i*1024:(i+1)*1024], file_inode.Zone[i])
		}
	} else if file_inode.File_size <= inode.First_size*1024 {
		for i := 0; i < 8; i++ {
			inode.Write_Block(file, data[i*1024:(i+1)*1024], file_inode.Zone[i])
		}

		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < whole-8; i++ {
			first_block_index := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			inode.Write_Block(file, data[i*1024:(i+1)*1024], first_block_index)
		}
	} else {
		for i := 0; i < 8; i++ {
			inode.Write_Block(file, data[i*1024:(i+1)*1024], file_inode.Zone[i])
		}
		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < 256; i++ {
			first_block_index := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			if first_block_index != 0 {
				inode.Write_Block(file, data[i*1024:(i+1)*1024], first_block_index)
			}
		}

		second_block := inode.Get_Block(file, file_inode.Zone[9])
		for i := 0; i < 256; i++ {
			breakFlag := false
			second_block_index := basic.BytesToInt(second_block[i*4 : (i+1)*4])
			if second_block_index != 0 {
				second_block_inside := inode.Get_Block(file, second_block_index)
				for j := 0; j < 256; j++ {
					second_block_inside_index := basic.BytesToInt(second_block_inside[j*4 : (j+1)*4])
					if second_block_inside_index != 0 {
						inode.Write_Block(file, data[i*1024:(i+1)*1024], second_block_inside_index)
					} else {
						breakFlag = true
						break
					}
				}
			}
			if breakFlag {
				break
			}
		}
	}
}

// 因为这两个函数没什么意思，重复度也比较高，不再进行无意义的注释，只保留关键或者独有部分
func Read_data(file *os.File, file_inode inode.INode) (data []byte) {

	// 几个块
	whole := file_inode.File_size

	var zone_list []int
	if file_inode.File_size <= inode.Direct_size*1024 {
		for i := 0; i < whole; i++ {
			zone_list = append(zone_list, file_inode.Zone[i])
		}
	} else if file_inode.File_size <= inode.First_size*1024 {
		for i := 0; i < 8; i++ {
			zone_list = append(zone_list, file_inode.Zone[i])
		}

		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < whole-8; i++ {
			first_block_index := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			zone_list = append(zone_list, first_block_index)
		}
	} else {
		for i := 0; i < 8; i++ {
			zone_list = append(zone_list, file_inode.Zone[i])
		}
		first_block := inode.Get_Block(file, file_inode.Zone[8])
		for i := 0; i < 256; i++ {
			first_block_index := basic.BytesToInt(first_block[i*4 : (i+1)*4])
			if first_block_index != 0 {
				zone_list = append(zone_list, first_block_index)
			} else {
				break
			}

		}

		second_block := inode.Get_Block(file, file_inode.Zone[9])
		for i := 0; i < 256; i++ {
			breakFlag := false
			second_block_index := basic.BytesToInt(second_block[i*4 : (i+1)*4])
			if second_block_index != 0 {
				second_block_inside := inode.Get_Block(file, second_block_index)
				for j := 0; j < 256; j++ {
					second_block_inside_index := basic.BytesToInt(second_block_inside[j*4 : (j+1)*4])
					if second_block_inside_index != 0 {
						zone_list = append(zone_list, second_block_inside_index)
					} else {
						breakFlag = true
						break
					}
				}
			}
			if breakFlag {
				break
			}
		}
	}

	// 开始向外读取，一个byte一个byte
	for i := 0; i < len(zone_list); i++ {
		if len(zone_list) != 1 && i != len(zone_list)-1 {
			for j := 0; j < 1024; j++ {
				data = append(data, inode.Get_Block(file, zone_list[i])[j])
			}
		} else {
			not_whole_block := inode.Get_Block(file, zone_list[i])
			for j := 0; j < 1024; j++ {
				if not_whole_block[j] != 0 {
					data = append(data, not_whole_block[j])
				} else {
					break
				}

			}
		}

	}

	return data
}

func RWX_JUDGE(now inode.INode, cmd string) bool {
	if now.Filetype == "dir" { // 是目录

		if cmd == "dir" { // dir命令，r权限
			if boot.Working_User.Uid == now.Owner_id {
				if now.Control[0]/4 == 1 {
					return true
				} else {
					return false
				}
			} else if boot.Working_User.Gid == now.Group_id {
				if now.Control[1]/4 == 1 {
					return true
				} else {
					return false
				}
			} else {
				if now.Control[2]/4 == 1 {
					return true
				} else {
					return false
				}

			}
		} else if cmd == "update" { // 修改，没有专门命令，w权限

			if boot.Working_User.Uid == now.Owner_id {

				if now.Control[0]/2%2 == 1 {
					return true
				} else {
					return false
				}
			} else if boot.Working_User.Gid == now.Group_id {
				if now.Control[1]/2%2 == 1 {
					return true
				} else {
					return false
				}
			} else {
				if now.Control[2]/2%2 == 1 {
					return true
				} else {
					return false
				}
			}
		} else if cmd == "cd" { // cd命令，x权限
			if boot.Working_User.Uid == now.Owner_id {
				if now.Control[0]%2 == 1 {
					return true
				} else {
					return false
				}
			} else if boot.Working_User.Gid == now.Group_id {
				if now.Control[1]%2 == 1 {
					return true
				} else {
					return false
				}
			} else {
				if now.Control[2]%2 == 1 {
					return true
				} else {
					return false
				}

			}
		}
	} else { // 对于文件
		if cmd == "read" { // 读取，没有专门命令，r权限
			if boot.Working_User.Uid == now.Owner_id {
				if now.Control[0]/4 == 1 {
					return true
				} else {
					return false
				}
			} else if boot.Working_User.Gid == now.Group_id {
				if now.Control[1]/4 == 1 {
					return true
				} else {
					return false
				}
			} else {
				if now.Control[2]/4 == 1 {
					return true
				} else {
					return false
				}

			}
		} else if cmd == "update" { // 修改，没有专门命令，w权限
			if boot.Working_User.Uid == now.Owner_id {
				if now.Control[0]/2%2 == 1 {
					return true
				} else {
					return false
				}
			} else if boot.Working_User.Gid == now.Group_id {
				if now.Control[1]/2%2 == 1 {
					return true
				} else {
					return false
				}
			} else {
				if now.Control[2]/2%2 == 1 {
					return true
				} else {
					return false
				}
			}
		} else if cmd == "exec" { // 执行，没有专门命令，x权限
			if boot.Working_User.Uid == now.Owner_id {
				if now.Control[0]%2 == 1 {
					return true
				} else {
					return false
				}
			} else if boot.Working_User.Gid == now.Group_id {
				if now.Control[1]%2 == 1 {
					return true
				} else {
					return false
				}
			} else {
				if now.Control[2]%2 == 1 {
					return true
				} else {
					return false
				}

			}
		}
	}
	return false
}

func Get_target_dir_inode(file *os.File, target_dir string) (target_dir_inode inode.INode) {
	now_dir_back := boot.Working_Directory
	now_dir_inode_back := boot.Working_INode
	now_dir_dentry_back := boot.Working_Dentry

	// 无视权限的cd，原因是只需要获取inode，不需要在其中执行东西
	Cd(file, []string{target_dir}, 1)
	target_dir_inode = inode.Get_INode(file, boot.Working_INode)

	boot.Working_Directory = now_dir_back
	boot.Working_INode = now_dir_inode_back
	boot.Working_Dentry = now_dir_dentry_back
	return target_dir_inode
}

func Update(file *os.File) {
	// boot块和超级块不用管

	// imap
	imap.IB = imap.Get_IMap_Block(file)
	imap.Free_INode = imap.IB.Free_INode
	imap.Last_position = imap.IB.Last_position
	imap.Get_IMap(file)

	// zmap
	zmap.ZB = zmap.Get_ZMap_Block(file)
	zmap.Free_Block = zmap.ZB.Free_Block
	zmap.Last_position = zmap.ZB.Last_position
	zmap.Disk_size_remain = zmap.ZB.Disk_size_remain
	zmap.Get_ZMap(file)

	// inode
	inode.Root_INode = inode.Get_INode(file, 0)
	inode.Root_Dentry = inode.Get_Dentry(file, 0)
}
