package watch

import "time"

type SateFile struct {
	Watch    Watch    `json:"watch"`
	Obsolete []string `json:"obsolete"`
}

type Watch struct {
	FileInfoList []map[string]FileInfo `json:"file_info_list"`
}

// type FileInfoList map[string]FileInfo

type FileInfo struct {
	Path         string    `json:"path"`           // 文件地址
	Offset       int64     `json:"offset"`         // 当前文件读取的偏移量
	LastReadTime time.Time `json:"last_read_time"` // 最后一次读取文件的时间
}
