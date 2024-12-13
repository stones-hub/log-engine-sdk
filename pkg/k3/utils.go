package k3

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func InArray(slice []string, item string) bool {

	for _, v := range slice {
		if v == item {
			return true
		}
	}

	return false
}

func parseTime(input []byte) string {
	var re = regexp.MustCompile(`"((\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})(?:\.(\d{3}))\d*)(Z|[\+-]\d{2}:\d{2})"`)
	var substitution = "\"$2 $3.$4\""

	for re.Match(input) {
		input = re.ReplaceAll(input, []byte(substitution))
	}
	return string(input)
}

// 对字符串做压缩
func CompressGzip(data string) (string, error) {
	var (
		buf bytes.Buffer
		err error
		w   *gzip.Writer
	)

	w = gzip.NewWriter(&buf)

	if _, err = w.Write([]byte(data)); err != nil {
		return "", err
	}

	w.Close()

	return buf.String(), nil
}

func DecompressGzip(data string) (string, error) {
	var (
		buf              bytes.Buffer
		err              error
		r                *gzip.Reader
		decompressedData []byte
	)

	buf.Write([]byte(data))
	if r, err = gzip.NewReader(&buf); err != nil {
		return "", err
	}

	if decompressedData, err = io.ReadAll(r); err != nil {
		return "", err
	}

	return string(decompressedData), nil
}

func MergeProperties(target, source map[string]interface{}) {
	for k, v := range source {
		target[k] = v
	}
}

func GetLocalIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, addr := range addrs {
		// 检查是否为 IP net.Addr 类型
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		// 获取 IP 地址
		ip := ipNet.IP

		// 排除回环地址
		if ip.IsLoopback() {
			continue
		}

		// 添加 IP 地址
		if ip.To4() != nil || ip.To16() != nil {
			ips = append(ips, ip.String())
		}
	}

	return ips, nil
}

func GenerateUUID() string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return ""
	}
	return newUUID.String()
}

// FetchDirectory 递归遍历目录, maxDepth -1 全部遍历, 返回所有的文件
func FetchDirectory(dir string, maxDepth int) ([]string, error) {

	var (
		files     []string
		rootDepth = len(strings.Split(dir, string(os.PathSeparator)))
		err       error
	)

	// fmt.Println(files, rootDepth, string(os.PathSeparator))
	if err = filepath.WalkDir(dir, func(currentPath string, d os.DirEntry, err error) error {
		var (
			currentDepth int
		)

		if err != nil {
			return err
		}

		currentDepth = len(strings.Split(currentPath, string(os.PathSeparator))) - rootDepth

		if maxDepth != -1 && currentDepth > maxDepth {
			return filepath.SkipDir // 跳过当前目录
		}

		if !d.IsDir() {
			files = append(files, currentPath)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

// FetchDirectoryPath 递归遍历目录, maxDepth -1 全部遍历, 返回所有的目录
func FetchDirectoryPath(dir string, maxDepth int) ([]string, error) {
	var (
		err       error
		paths     []string
		rootDepth = len(strings.Split(dir, string(os.PathSeparator)))
	)

	if err = filepath.WalkDir(dir, func(currentPath string, d fs.DirEntry, err error) error {
		var (
			currentDepth int
		)

		if err != nil {
			return err
		}

		currentDepth = len(strings.Split(currentPath, string(os.PathSeparator))) - rootDepth

		if maxDepth != -1 && currentDepth > maxDepth {
			return filepath.SkipDir
		}

		if d.IsDir() {
			paths = append(paths, currentPath)
		}

		return nil

	}); err != nil {
		return nil, err
	}

	return paths, nil
}

func InterfaceToString(val interface{}) (string, bool) {
	if val == nil {
		return "", false
	}
	strVal, ok := val.(string)
	return strVal, ok
}

func InterfaceToJSONString(val interface{}) (string, error) {
	b, err := json.Marshal(val)
	return string(b), err
}

// RemoveDuplicateElement 移除重复元素
func RemoveDuplicateElement(src []string) []string {
	var (
		seen = make(map[string]bool)
		res  []string
	)

	for _, s := range src {
		if _, exists := seen[s]; !exists {
			seen[s] = true
			res = append(res, s)
		}
	}

	return res
}

func InSlice(s string, slice []string) bool {
	for _, d := range slice {
		if s == d && len(s) == len(d) {
			return true
		}
	}
	return false
}

// FileExists 判断文件是否存在
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
