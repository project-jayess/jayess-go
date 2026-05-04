package runtime

type FilesystemCapability struct {
	Name          string
	RuntimeSymbol string
	Kind          string
}

func FilesystemCapabilities() []FilesystemCapability {
	return []FilesystemCapability{
		{Name: "readFile", RuntimeSymbol: "jayess_fs_read_file", Kind: "function"},
		{Name: "writeFile", RuntimeSymbol: "jayess_fs_write_file", Kind: "function"},
		{Name: "appendFile", RuntimeSymbol: "jayess_fs_append_file", Kind: "function"},
		{Name: "deleteFile", RuntimeSymbol: "jayess_fs_delete_file", Kind: "function"},
		{Name: "rename", RuntimeSymbol: "jayess_fs_rename", Kind: "function"},
		{Name: "copyFile", RuntimeSymbol: "jayess_fs_copy_file", Kind: "function"},
		{Name: "stat", RuntimeSymbol: "jayess_fs_stat", Kind: "function"},
		{Name: "chmod", RuntimeSymbol: "jayess_fs_chmod", Kind: "function"},
		{Name: "exists", RuntimeSymbol: "jayess_fs_exists", Kind: "function"},
		{Name: "mkdir", RuntimeSymbol: "jayess_fs_mkdir", Kind: "function"},
		{Name: "mkdirp", RuntimeSymbol: "jayess_fs_mkdirp", Kind: "function"},
		{Name: "rmdir", RuntimeSymbol: "jayess_fs_rmdir", Kind: "function"},
		{Name: "readdir", RuntimeSymbol: "jayess_fs_readdir", Kind: "function"},
		{Name: "walkDir", RuntimeSymbol: "jayess_fs_walk_dir", Kind: "function"},
		{Name: "symlink", RuntimeSymbol: "jayess_fs_symlink", Kind: "function"},
		{Name: "watch", RuntimeSymbol: "jayess_fs_watch", Kind: "function"},
		{Name: "createReadStream", RuntimeSymbol: "jayess_fs_create_read_stream", Kind: "function"},
		{Name: "createWriteStream", RuntimeSymbol: "jayess_fs_create_write_stream", Kind: "function"},
	}
}

func HasFilesystemCapability(name string) bool {
	for _, capability := range FilesystemCapabilities() {
		if capability.Name == name {
			return capability.RuntimeSymbol != ""
		}
	}
	return false
}
