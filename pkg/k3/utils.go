package k3

import (
	"bytes"
	"compress/gzip"
	"github.com/google/uuid"
	"io"
	"net"
	"regexp"
)

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
