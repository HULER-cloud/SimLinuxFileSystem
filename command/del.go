package command

import (
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/imap"
	"golangsfs/in_out"
	"golangsfs/inode"
	"golangsfs/zmap"
	"os"
)

func Del(file *os.File, args []string) {
	// 权限控制1
	if boot.Working_User.Uid != "0" {
		target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
		if !RWX_JUDGE(target_dir_inode, "w") {
			in_out.Out("当前用户权限不足，无法在该目录删除文件！")
			return
		}
	}
	target_file := args[0]
	// 老生常谈，获取dentry
	inode_list := inode.Get_Dentry(file, boot.Working_Dentry).Content
	if len(inode_list) == 0 {
		return
	}
	// 遍历
	for i := 0; i < len(inode_list); i++ {
		temp_inode := inode.Get_INode(file, inode_list[i])
		if temp_inode.Filetype == "dir" {
			continue
		}
		fullname := temp_inode.Filename
		if temp_inode.Filetype != "" {
			fullname += ("." + temp_inode.Filetype)
		}
		// 说明找到了要删除的文件
		if target_file == fullname {

			// 权限控制2
			if boot.Working_User.Uid != "0" {
				target_file_inode := temp_inode
				if !RWX_JUDGE(target_file_inode, "w") {
					in_out.Out("当前用户权限不足，无法删除该文件！")
					return
				}
			}

			// 记录要清除的文件块
			var to_be_cleaned []int

			// 直接块
			if temp_inode.File_size <= inode.Direct_size {
				for j := 0; j < 8; j++ {
					if temp_inode.Zone[j] != 0 {
						// 一个文件的文件块之间不一定是连续的，所以得一个一个记录，后面清除
						to_be_cleaned = append(to_be_cleaned, temp_inode.Zone[j])
					}
				}
			} else if temp_inode.File_size <= inode.First_size { // 一级间接块
				// 直接块全部记录
				for j := 0; j < 8; j++ {
					to_be_cleaned = append(to_be_cleaned, temp_inode.Zone[j])
				}

				data := inode.Get_Block(file, temp_inode.Zone[8])
				size_remain := temp_inode.File_size - inode.Direct_size
				// 遍历剩余空间
				for j := 0; j < size_remain; j++ {
					extra_block := basic.BytesToInt(data[j*4 : (j+1)*4])
					if extra_block != 0 {
						to_be_cleaned = append(to_be_cleaned, extra_block)
					} else {
						break
					}
				}
				to_be_cleaned = append(to_be_cleaned, temp_inode.Zone[8])
			} else {
				// 直接块全部记录
				for j := 0; j < 8; j++ {
					to_be_cleaned = append(to_be_cleaned, temp_inode.Zone[j])
				}

				data := inode.Get_Block(file, temp_inode.Zone[8])
				// 一级间接块也全部记录
				for j := 0; j < 256; j++ {
					extra_block := basic.BytesToInt(data[j*4 : (j+1)*4])
					to_be_cleaned = append(to_be_cleaned, extra_block)
				}
				to_be_cleaned = append(to_be_cleaned, temp_inode.Zone[8])

				data = inode.Get_Block(file, temp_inode.Zone[9])
				for j := 0; j < 256; j++ {
					extra_block := basic.BytesToInt(data[j*4 : (j+1)*4])
					breakFlag := false
					// 二级间接块不一定用全，先检测着
					if extra_block != 0 {
						data_2 := inode.Get_Block(file, extra_block)
						for k := 0; k < 256; k++ {
							extra_block_2 := basic.BytesToInt(data_2[k*4 : (k+1)*4])
							// 有选择的记录
							if extra_block_2 != 0 {
								to_be_cleaned = append(to_be_cleaned, extra_block_2)
							} else {
								breakFlag = true
								break
							}
						}
						to_be_cleaned = append(to_be_cleaned, extra_block)
						if breakFlag {
							break
						}
					} else {
						break
					}
				}
				to_be_cleaned = append(to_be_cleaned, temp_inode.Zone[9])
			}
			// 开始清除
			for j := 0; j < len(to_be_cleaned); j++ {
				basic.Clean(file, to_be_cleaned[j], to_be_cleaned[j])
			}

			// 更新ZMap
			for block_index := range to_be_cleaned {
				byte_index := block_index / 8
				bit_index := block_index % 8
				zmap.ZMap[byte_index] = ^(zmap.ZMap[byte_index] & (1 << (7 - bit_index)))
				zmap.Write_ZMap(file)
				zmap.Free_Block += len(to_be_cleaned)
				zmap.Disk_size_remain += len(to_be_cleaned)
				zmap.ZB.Free_Block = zmap.Free_Block
				zmap.ZB.Disk_size_remain = zmap.Disk_size_remain
				zmap.Write_ZMap_Block(file, zmap.ZB)
			}
			// 清除inode
			basic.Clean(file, inode_list[i], inode_list[i])
			// 更新IMap
			byte_index := inode_list[i] / 8
			bit_index := inode_list[i] % 8
			imap.IMap[byte_index] = ^(imap.IMap[byte_index] & (1 << (7 - bit_index)))
			imap.Write_IMap(file)
			imap.Free_INode++
			imap.IB.Free_INode++
			imap.Write_IMap_Block(file, imap.IB)

			// 处理上级目录冗余信息
			temp_working_dir_dentry := inode.Get_Dentry(file, boot.Working_Dentry)
			for j := 0; j < len(temp_working_dir_dentry.Content); j++ {
				if temp_working_dir_dentry.Content[j] == inode_list[i] {
					temp_working_dir_dentry.Content =
						append(temp_working_dir_dentry.Content[:j],
							temp_working_dir_dentry.Content[j+1:]...)
					break
				}
			}

			inode.Write_Dentry(file, temp_working_dir_dentry, boot.Working_Dentry)
			// 更新内存中root数据
			if boot.Working_Directory == "/" {
				inode.Root_INode = inode.Get_INode(file, 0)
				inode.Root_Dentry = inode.Get_Dentry(file, 0)
			}
			return
		}
	}
}
