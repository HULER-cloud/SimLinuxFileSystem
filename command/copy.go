package command

import (
	"golangsfs/basic"
	"golangsfs/boot"
	"golangsfs/in_out"
	"golangsfs/inode"
	"os"
	"strings"
	"time"
)

func Copy(file *os.File, args []string) {
	// 从宿主机拷到simdisk，这里我们默认为覆写模式
	if strings.HasPrefix(args[0], "<host>") {
		// 先获取到宿主机要拷贝文件的文件名和文件类型
		host_filename := ""
		host_filetype := ""
		_, host_filename, host_filetype = basic.Analyse_file_path(args[0])

		// 切片排除<host>，直接得到目录信息
		host_file_data, err := os.ReadFile(args[0][6:])
		if err != nil {
			in_out.Out("宿主机目标文件不存在！")
			return
		}

		// 写入文件
		dir_inode, dir_dentry, isSuccess := Is_target_dir_exist(file, args[1])
		if isSuccess { // 目录存在
			// 权限控制
			if boot.Working_User.Uid != "0" {
				target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
				if !RWX_JUDGE(target_dir_inode, "w") {
					in_out.Out("当前用户权限不足，无法在该目录修改文件！")
					return
				}
			}
			// 预备更新inode中的修改时间
			curr_time := time.Now()
			file_inode, file_inode_index, isSuccess_2 := Is_target_file_exist(file, dir_dentry, host_filename, host_filetype)

			now_dir_back := boot.Working_Directory
			now_dir_inode_back := boot.Working_INode
			now_dir_dentry_back := boot.Working_Dentry

			Cd(file, args[1:], 0)

			if isSuccess_2 { // 文件也存在

				// 文件存在时权限控制
				if boot.Working_User.Uid != "0" {
					target_file_inode := file_inode
					if !RWX_JUDGE(target_file_inode, "w") {
						in_out.Out("当前用户权限不足，无法写入该文件！")
						return
					}
				}

				// 那就先清空再覆盖写入
				if file_inode.File_size != 0 {
					Del_zone(file, file_inode, file_inode_index)
				}

				file_inode = inode.Get_INode(file, file_inode_index)
				// 申请空间
				Allocate_zone(file, file_inode, file_inode_index, len(host_file_data))
				file_inode = inode.Get_INode(file, file_inode_index)
				// 写入数据
				Write_data(file, file_inode, host_file_data)
				// 父目录信息更改
				dir_inode.Modify_time = curr_time
				inode.Write_INode(file, dir_inode, dir_dentry.To_INode)

			} else { // 文件不存在

				// 权限控制
				if boot.Working_User.Uid != "0" {
					target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
					if !RWX_JUDGE(target_dir_inode, "w") {
						in_out.Out("当前用户权限不足，无法在该目录新建文件！")
						return
					}
				}

				// 那就先新建文件再写入
				if host_filetype == "" {
					file_inode, file_inode_index = Newfile(file, []string{host_filename})
				} else {
					file_inode, file_inode_index = Newfile(file, []string{host_filename + "." + host_filetype})
				}
				// 申请空间
				Allocate_zone(file, file_inode, file_inode_index, len(host_file_data))
				// 进行更新
				file_inode = inode.Get_INode(file, file_inode_index)
				// 写入数据
				Write_data(file, file_inode, host_file_data)

				// 父目录信息更改
				dir_inode.Modify_time = curr_time
				dir_dentry.Content = append(dir_dentry.Content, file_inode_index)

				inode.Write_INode(file, dir_inode, dir_dentry.To_INode)
				inode.Write_Dentry(file, dir_dentry, dir_inode.Zone[0])

			}

			boot.Working_Directory = now_dir_back
			boot.Working_INode = now_dir_inode_back
			boot.Working_Dentry = now_dir_dentry_back

			// 更新内存中root数据
			if boot.Working_Directory == "/" {
				inode.Root_INode = inode.Get_INode(file, 0)
				inode.Root_Dentry = inode.Get_Dentry(file, 0)
			}
		} else {
			in_out.Out("simdisk目标目录不存在！")
			return
		}

	} else if strings.HasPrefix(args[1], "<host>") {
		// 从simdisk拷到宿主机

		// 获取到simdisk这边的信息
		simdisk_dir, simdisk_filename, simdisk_filetype := basic.Analyse_file_path(args[0])

		in_out.Out(simdisk_dir + simdisk_filename + simdisk_filetype)

		// 将宿主机目录和simdisk这边的文件名拼在一起，方便后续打开
		host_path := args[1][6:]
		if host_path[len(host_path)-1] == '\\' || host_path[len(host_path)-1] == '/' {
			if simdisk_filetype == "" {
				host_path += (simdisk_filename)
			} else {
				host_path += (simdisk_filename + "." + simdisk_filetype)
			}

		} else {
			if simdisk_filetype == "" {
				host_path += ("\\" + simdisk_filename)
			} else {
				host_path += ("\\" + simdisk_filename + "." + simdisk_filetype)
			}
		}

		// 此时宿主机文件设置为只写模式，不存在则新建，写类型为覆盖
		host_file, err := os.OpenFile(host_path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)

		if err != nil {
			in_out.Out("宿主机目录不存在！")
			return
		}

		// simdisk目标文件目录是否存在
		// 先搞出来目录
		_, dir_dentry, isSuccess := Is_target_dir_exist(file, simdisk_dir)

		// 预备读数据
		var simdisk_file_data []byte

		if isSuccess { // 目录存在

			file_inode, _, isSuccess_2 := Is_target_file_exist(file, dir_dentry, simdisk_filename, simdisk_filetype)

			now_dir_back := boot.Working_Directory
			now_dir_inode_back := boot.Working_INode
			now_dir_dentry_back := boot.Working_Dentry

			Cd(file, []string{simdisk_dir}, 0)
			// 权限控制
			if boot.Working_User.Uid != "0" {
				target_file_inode := file_inode
				if !RWX_JUDGE(target_file_inode, "read") {
					in_out.Out("当前用户权限不足，无法读取该文件！")
					return
				}
			}

			if isSuccess_2 { // 文件也存在
				// 读出
				simdisk_file_data = Read_data(file, file_inode)
			} else { // 文件不存在
				in_out.Out("simdisk目标文件不存在！")
				return
			}

			boot.Working_Directory = now_dir_back
			boot.Working_INode = now_dir_inode_back
			boot.Working_Dentry = now_dir_dentry_back
		} else {
			in_out.Out("simdisk目标目录不存在！")
			return
		}

		// 最后写入宿主机目标文件
		host_file.Write(simdisk_file_data)

	} else {
		// simdisk内互相拷贝

		// 其实就是前面两个分支各取一半拼起来（大致上）
		simdisk_dir, simdisk_filename, simdisk_filetype := basic.Analyse_file_path(args[0])
		_, dir_dentry, isSuccess := Is_target_dir_exist(file, simdisk_dir)

		// 预备读取数据
		var simdisk_file_data []byte

		if isSuccess { // 目录存在

			file_inode, _, isSuccess_2 := Is_target_file_exist(file, dir_dentry, simdisk_filename, simdisk_filetype)

			now_dir_back := boot.Working_Directory
			now_dir_inode_back := boot.Working_INode
			now_dir_dentry_back := boot.Working_Dentry

			Cd(file, []string{simdisk_dir}, 0)

			// 权限控制
			if boot.Working_User.Uid != "0" {
				target_file_inode := file_inode
				if !RWX_JUDGE(target_file_inode, "read") {
					in_out.Out("当前用户权限不足，无法读取该文件！")
					return
				}
			}

			if isSuccess_2 { // 文件也存在
				// 读出
				simdisk_file_data = Read_data(file, file_inode)
			} else { // 文件不存在
				in_out.Out("simdisk源文件不存在！")
				return
			}

			boot.Working_Directory = now_dir_back
			boot.Working_INode = now_dir_inode_back
			boot.Working_Dentry = now_dir_dentry_back
		} else {
			in_out.Out("simdisk源目录不存在！")
			return
		}

		dir_inode2, dir_dentry2, isSuccess2 := Is_target_dir_exist(file, args[1])
		if isSuccess2 { // 目录存在

			// 权限控制
			if boot.Working_User.Uid != "0" {
				target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
				if !RWX_JUDGE(target_dir_inode, "w") {
					in_out.Out("当前用户权限不足，无法在该目录修改文件！")
					return
				}
			}

			// 可能的修改
			curr_time := time.Now()
			file_inode, file_inode_index, isSuccess_inside := Is_target_file_exist(file, dir_dentry2, simdisk_filename, simdisk_filetype)

			now_dir_back := boot.Working_Directory
			now_dir_inode_back := boot.Working_INode
			now_dir_dentry_back := boot.Working_Dentry

			Cd(file, args[1:], 0)

			if isSuccess_inside { // 文件也存在
				// 文件存在时权限控制
				if boot.Working_User.Uid != "0" {
					target_file_inode := file_inode
					if !RWX_JUDGE(target_file_inode, "w") {
						in_out.Out("当前用户权限不足，无法写入该文件！")
						return
					}
				}

				// 那就先清空再覆盖写入
				Del_zone(file, file_inode, file_inode_index)

				file_inode = inode.Get_INode(file, file_inode_index)
				// 申请空间
				Allocate_zone(file, file_inode, file_inode_index, len(simdisk_file_data))
				file_inode = inode.Get_INode(file, file_inode_index)
				// 写入数据
				Write_data(file, file_inode, simdisk_file_data)
				// 父目录信息更改
				dir_inode2.Modify_time = curr_time

				inode.Write_INode(file, dir_inode2, dir_dentry.To_INode)
				inode.Write_Dentry(file, dir_dentry2, dir_inode2.Zone[0])

			} else { // 文件不存在

				// 权限控制
				if boot.Working_User.Uid != "0" {
					target_dir_inode := Get_target_dir_inode(file, boot.Working_Directory)
					if !RWX_JUDGE(target_dir_inode, "w") {
						in_out.Out("当前用户权限不足，无法在该目录新建文件！")
						return
					}
				}

				// 那就先新建文件再写入
				if simdisk_filetype == "" {
					file_inode, file_inode_index = Newfile(file, []string{simdisk_filename})
				} else {
					file_inode, file_inode_index = Newfile(file, []string{simdisk_filename + "." + simdisk_filetype})
				}
				// 申请空间
				Allocate_zone(file, file_inode, file_inode_index, len(simdisk_file_data))
				// 进行更新
				file_inode = inode.Get_INode(file, file_inode_index)
				// 写入数据
				Write_data(file, file_inode, simdisk_file_data)

				// 父目录信息更改
				dir_inode2.Modify_time = curr_time
				dir_dentry2.Content = append(dir_dentry2.Content, file_inode_index)

				inode.Write_INode(file, dir_inode2, dir_dentry2.To_INode)
				inode.Write_Dentry(file, dir_dentry2, dir_inode2.Zone[0])

			}

			boot.Working_Directory = now_dir_back
			boot.Working_INode = now_dir_inode_back
			boot.Working_Dentry = now_dir_dentry_back

			// 更新内存中root数据
			if boot.Working_Directory == "/" {
				inode.Root_INode = inode.Get_INode(file, 0)
				inode.Root_Dentry = inode.Get_Dentry(file, 0)
			}
		} else {
			in_out.Out("目标目录不存在！")
			return
		}

	}
}
