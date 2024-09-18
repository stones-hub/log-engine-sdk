package watch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/protocol"
	"log-engine-sdk/pkg/k3/sender"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	GlobalWatchSg                = &sync.WaitGroup{}
	GlobalWatchClose             = make(chan struct{})
	GlobalWatcher                *fsnotify.Watcher
	GlobalForceSyncStateFileChan = make(chan struct{})

	// GlobalFileStates 用于存储文件读取状态
	GlobalFileStates   = make(map[string]*FileState)
	GlobalFileStatesFd = make(map[string]*FileSateFd)
	// GlobalSateFileLock 用于解决stateFile文件读写的锁
	GlobalSateFileLock sync.Mutex
)

var dataAnalytics k3.DataAnalytics

func InitConsumerBatchLog() error {
	var (
		elk      *sender.ELKServer
		err      error
		consumer protocol.K3Consumer
	)
	if elk, err = sender.NewELKServer([]string{"http://127.0.0.1:9200"}, "admin", "admin", ""); err != nil {
		return err
	}

	if consumer, err = k3.NewBatchConsumerWithConfig(k3.K3BatchConsumerConfig{
		Sender:    elk,
		AutoFlush: true,
	}); err != nil {
		return err
	}
	dataAnalytics = k3.NewDataAnalytics(consumer)
	return nil
}

// InitWatcher 监听指定目录下的文件变化
func InitWatcher(paths []string, stateFile string) (*fsnotify.Watcher, error) {
	var (
		err error
	)

	if GlobalWatcher, err = fsnotify.NewWatcher(); err != nil {
		k3.K3LogError("WatchDirectory: %s", err)
		return nil, err
	}

	GlobalWatchSg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				k3.K3LogError("InitWatcher Recover: %s", r)
			}
			GlobalWatchSg.Done()
			k3.ForceExit()
		}()

		for {
			select {
			case event, ok := <-GlobalWatcher.Events:
				if !ok {
					k3.K3LogError("WatchEvents error : %v", ok)
					return
				}
				handleEvent(event, stateFile)

			case err, ok := <-GlobalWatcher.Errors:
				if !ok {
					k3.K3LogError("WatchErrors error : %v, %v", err, ok)
					return
				}
				k3.K3LogError("WatchErrors: %s", err)

			case <-GlobalWatchClose:
				k3.K3LogInfo("Watcher been Closed.")
				return

				// 强制同步一次state file
			case <-GlobalForceSyncStateFileChan:
				if err := SyncToSateFile(stateFile); err != nil {
					k3.K3LogError("SyncToSateFile error: %s", err)
				}
			}
		}
	}()

	// 初始化时，需要监控所有的子目录, 递归遍历目录下的所有子目录
	for _, dir := range paths {
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// 添加子目录到监视器
				k3.K3LogInfo("Adding directory to watch: %s", path)
				err = GlobalWatcher.Add(path)
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return GlobalWatcher, nil
}

// FileState 文件读取状态 存储在文件中
type FileState struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
}

type FileSateFd struct {
	Fd *os.File
}

func Run(directorys []string, stateFile string) error {
	var (
		err       error
		fileNames []string
		fileName  string
		state     *FileState
		file      *os.File
	)

	k3.K3LogInfo("WatchDirectory: %s, StateFile: %s", directorys, stateFile)

	if err = InitConsumerBatchLog(); err != nil {
		return err
	}

	// 1. 判断stateFile是否存在，如果不存在，则创建一个空的stateFile, 如果存在，直接返回
	if err = GenerateEmptyFile(stateFile); err != nil {
		return err
	}

	// 2. 将stateFile中的内容映射到GlobalFileStates
	if err = LoadStateFile(stateFile); err != nil {
		return err
	}

	// 3. 遍历WatchDirectory下的所有文件
	for _, dir := range directorys {
		if names, err := k3.FetchDirectory(dir, -1); err != nil {
			return err
		} else {
			fileNames = append(fileNames, names...)
		}
	}

	// 4.清理掉目录中已经不存在的文件
	for fileName, _ = range GlobalFileStates {
		if k3.InArray(fileNames, fileName) == false {
			delete(GlobalFileStates, fileName)
		}
	}

	// 5. 将新增的文件补充到filestates
	for _, fileName = range fileNames {
		if _, ok := GlobalFileStates[fileName]; !ok {
			GlobalFileStates[fileName] = &FileState{
				Path:   fileName,
				Offset: 0,
			}
		}
	}

	// 6. 统一保存一次fileStates, 会先清空在保存
	if err = SyncToSateFile(stateFile); err != nil {
		return err
	}

	// 7. 读取所有文件的fd，存储起来
	for _, state = range GlobalFileStates {
		if file, err = os.OpenFile(state.Path, os.O_RDONLY, 0644); err != nil {
			return fmt.Errorf("open file error: %s", err)
		}
		GlobalFileStatesFd[state.Path] = &FileSateFd{
			Fd: file,
		}
	}

	k3.K3LogInfo("数据初始化完毕: \n GlobalFileStates: %+v \n\n GlobalFileStatesFd: %+v\n", GlobalFileStates, GlobalFileStatesFd)

	// 8. 监听watchDirectory下的文件变化，当文件发生变化时，读取文件内容，直到最后一个\n 结束， 最后更新fileStates和stateFile
	if _, err = InitWatcher(directorys, stateFile); err != nil {
		Clean()
		return err
	}

	ClockUpdateStateFile()

	return nil
}

// Clean 关闭所有资源, 其中包含 watch和数据收集器的
func Clean() {
	// 终止watch进程
	close(GlobalWatchClose)
	// 等待watch进程结束
	k3.K3LogInfo("Waiting for watcher exit...")
	GlobalWatchSg.Wait()
	// 关闭watcher
	if GlobalWatcher != nil {
		k3.K3LogInfo("Watcher been closed success.")
		GlobalWatcher.Close()
	}

	// 关闭所有打开的文件
	for _, fileStatesFd := range GlobalFileStatesFd {
		k3.K3LogInfo("Close file: %s", fileStatesFd.Fd.Name())
		fileStatesFd.Fd.Close()
	}

	// 关闭数据收集器, 并将所有内存数据提交给日志集群
	dataAnalytics.Close()
}

// handleEvent 处理文件事件
func handleEvent(event fsnotify.Event, stateFile string) {

	if event.Op&fsnotify.Write == fsnotify.Write { // 写入

		if GlobalFileStatesFd[event.Name] == nil {
			k3.K3LogError("handleEvent write error : %s", event.Name)
			return
		}
		offset, err := readFileByOffset(GlobalFileStatesFd[event.Name].Fd, GlobalFileStates[event.Name].Offset)
		if err != nil {
			k3.K3LogError("ReadFileByOffset error: %s", err)
		}
		updateFileStateOffset(event.Name, offset)

	} else if event.Op&fsnotify.Create == fsnotify.Create { // 创建

		// 判断是否是目录，如果是目录需要添加监控
		if fileInfo, err := os.Stat(event.Name); err != nil {
			k3.K3LogError("WatchCreate Stat error: %s", err)
			return
		} else {
			if fileInfo.IsDir() && GlobalWatcher != nil {
				GlobalWatcher.Add(event.Name)
				k3.K3LogInfo("WatchCreate add watch path: %s", event.Name)
				return
			}
		}

		// 添加到fileStates
		if _, ok := GlobalFileStates[event.Name]; !ok {
			GlobalFileStates[event.Name] = &FileState{
				Path:   event.Name,
				Offset: 0,
			}
		}

		// 添加到fileStatesFd
		if _, ok := GlobalFileStatesFd[event.Name]; !ok {
			if file, err := os.OpenFile(event.Name, os.O_RDONLY, 0644); err != nil {
				k3.K3LogError("WatchCreate OpenFile error: %s", err)
			} else {
				GlobalFileStatesFd[event.Name] = &FileSateFd{
					Fd: file,
				}
			}
		}

		if err := SyncToSateFile(stateFile); err != nil {
			k3.K3LogError("WatchCreate SyncToSateFile error: %s", err)
		}

	} else if event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Remove == fsnotify.Remove { // 重命名
		doRemoveAndRenameEvent(event.Name, stateFile)
	}
}

func doRemoveAndRenameEvent(name string, stateFile string) {

	// 哪怕报错。保险点，都剔除监控，因为已经删除了， 没办法判断是不是目录
	if err := GlobalWatcher.Remove(name); err != nil {
		k3.K3LogError("WatchRemove Remove watch [%s] error: %s", name, err)
	}

	// 关闭文件
	if _, ok := GlobalFileStatesFd[name]; ok {
		GlobalFileStatesFd[name].Fd.Close()
		delete(GlobalFileStatesFd, name)
	}

	// 删除文件
	if _, ok := GlobalFileStates[name]; ok {
		delete(GlobalFileStates, name)
	}

	// 更新状态文件
	if err := SyncToSateFile(stateFile); err != nil {
		k3.K3LogError("WatchRemove SyncToSateFile error: %s", err)
	}

}

// ReadFileByOffset 读取文件内容， 从offset开始，直到遇到\n, 返回读取后，最后的偏移量
func readFileByOffset(fd *os.File, offset int64) (int64, error) {
	var (
		err              error
		reader           *bufio.Reader
		content          string
		newOffset        int64
		currentReadIndex int // 当前读取次数
	)
	currentReadIndex = 0
	newOffset = offset

	// 移动到offset位置, 从文件开始移动
	if _, err = fd.Seek(offset, io.SeekStart); err != nil {
		return newOffset, fmt.Errorf("seek file error: %s", err)
	}

	reader = bufio.NewReader(fd)

	for currentReadIndex <= config.GlobalConfig.System.MaxReadCount {

		// 控制最多读取次数
		currentReadIndex++

		line, err := reader.ReadString('\n')

		if err != nil && err != io.EOF {
			k3.K3LogError("ReadFileByOffset: %s, ReadString error: %s", fd.Name())
			goto EXIT
		}

		if err == io.EOF && len(line) > 0 && strings.HasSuffix(line, "\n") {
			k3.K3LogInfo("ReadFileByOffset: %s, EOF", fd.Name())
			content += line
			newOffset += int64(len(line))
			goto EXIT
		}

		if len(line) > 0 && strings.HasSuffix(line, "\n") {
			content += line
			newOffset += int64(len(line))
			continue
		}

		break
	}

EXIT:
	// k3.K3LogInfo("ReadFileByOffset: %s, content: %s", fd.Name(), content)
	sendDataToConsumer(content)
	return newOffset, nil
}

// sendDataToConsumer 将数据发送到数据收集器
func sendDataToConsumer(content string) {
	var (
		// TODO 后期可以考虑将参数作为配置文件
		accountId string
		appId     string
		eventName string
		uuid      string
		ip        string
		datas     []string
	)

	accountId = "yelei@3k.com"
	appId = "center.3k.com"
	eventName = "admin_log"
	uuid = k3.GenerateUUID()
	if ips, err := k3.GetLocalIPs(); err != nil {
		k3.K3LogError("GetLocalIPs error: %s", err)
		ip = "10.10.10.1"
	} else {
		if len(ips) >= 1 {
			ip = ips[0]
		}
	}
	datas = strings.Split(content, "\n")

	for _, data := range datas {
		data = strings.TrimSpace(data)
		data = strings.Trim(data, "\n")

		if len(data) == 0 {
			continue
		}

		if err := dataAnalytics.Track(accountId, appId, eventName, uuid, ip, map[string]interface{}{
			"data": data,
		}); err != nil {
			k3.K3LogError("sendDataToConsumer error: %s", err)
		}
	}

}

func updateFileStateOffset(fileName string, offset int64) {
	GlobalFileStates[fileName].Offset = offset
}

// GenerateEmptyFile 如果没有就创建一个空的文件
func GenerateEmptyFile(filePath string) error {
	var (
		err error
		fd  *os.File
	)

	if _, err = os.Stat(filePath); err != nil && os.IsNotExist(err) {
		// 如果目录不存在就创建
		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("create dir error: %s", err)
		}

		// 创建文件
		if fd, err = os.Create(filePath); err != nil {
			return fmt.Errorf("create file error: %s", err)
		}

		defer fd.Close()

	} else if err != nil {
		return err
	}

	return nil
}

func LoadStateFile(filePath string) error {
	var (
		fileInfo os.FileInfo
		err      error
		fd       *os.File
		decoder  *json.Decoder
	)

	if fileInfo, err = os.Stat(filePath); err != nil {
		return err
	} else {
		if fileInfo.Size() > 0 {

			if fd, err = os.OpenFile(filePath, os.O_RDWR, 0666); err != nil {
				return fmt.Errorf("open state file error: %s", err)
			}
			defer fd.Close()

			decoder = json.NewDecoder(fd)
			if err = decoder.Decode(&GlobalFileStates); err != nil {
				return fmt.Errorf("load state file error: %s", err)
			}
		}
		return nil
	}
}

// SyncToSateFile 将GlobalFileStates同步到stateFile, 先清空在写入, 覆盖
func SyncToSateFile(filePath string) error {
	var (
		fd      *os.File
		err     error
		encoder *json.Encoder
	)

	GlobalSateFileLock.Lock()
	defer GlobalSateFileLock.Unlock()

	if fd, err = os.OpenFile(filePath, os.O_RDWR|os.O_TRUNC, 0666); err != nil {
		return fmt.Errorf("open state file error: %s", err)
	}

	defer fd.Close()

	encoder = json.NewEncoder(fd)
	if err = encoder.Encode(&GlobalFileStates); err != nil {
		return fmt.Errorf("sync state file error: %s", err)
	}

	return nil
}

// ClockUpdateStateFile 定时器，定时更新statefile
func ClockUpdateStateFile() {

	var ticker = time.NewTicker(60 * time.Second)

	go func() {
		defer func() {
			ticker.Stop()
		}()

		for {
			<-ticker.C
			k3.K3LogInfo("Clock: %s, update state file.", time.Now().Format("2006-01-02 15:04:05"))
			GlobalForceSyncStateFileChan <- struct{}{}
		}
	}()
}
