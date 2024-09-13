package watch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"log-engine-sdk/pkg/k3"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	GlobalWatchSg    *sync.WaitGroup
	GlobalWatchClose = make(chan struct{})
	GlobalWatcher    *fsnotify.Watcher

	// 用于存储文件读取状态
	GlobalFileStates   = make(map[string]*FileState)
	GlobalFileStatesFd = make(map[string]*FileSateFd)
	// 用于解决stateFile文件读写的锁
	GlobalSateFileLock sync.Mutex
)

// InitWatcher 监听指定目录下的文件变化
func InitWatcher(path string) (*fsnotify.Watcher, error) {
	var (
		err error
	)
	GlobalWatchSg = &sync.WaitGroup{}

	if GlobalWatcher, err = fsnotify.NewWatcher(); err != nil {
		k3.K3LogError("WatchDirectory: %s", err)
		return nil, err
	}

	GlobalWatchSg.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				k3.K3LogError("WatchDirectory Recover: %s", r)
			}

			GlobalWatchSg.Done()
		}()

		for {
			select {
			case event, ok := <-GlobalWatcher.Events:
				if !ok {
					k3.K3LogError("WatchEvents error : %v \n", ok)
					return
				}

				handleEvent(event)
			case err, ok := <-GlobalWatcher.Errors:
				if !ok {
					k3.K3LogError("WatchErrors error : %v \n", ok)
					return
				}

				k3.K3LogError("WatchErrors: %s", err)

			case <-GlobalWatchClose:
				k3.K3LogInfo("Watcher been Closed.")
				return
			}
		}
	}()

	if err = GlobalWatcher.Add(path); err != nil {
		return GlobalWatcher, err
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

func Run(directory string, stateFile string) error {
	var (
		err       error
		fileNames []string
		fileName  string
		state     *FileState
		file      *os.File
	)

	k3.K3LogInfo("WatchDirectory: %s, StateFile: %s", directory, stateFile)

	// 1. 判断stateFile是否存在，如果不存在，则创建一个空的stateFile, 如果存在，则读取stateFile
	if err = GenerateEmptyFile(stateFile); err != nil {
		return err
	}

	// 2. 将读取的stateFile文件的内容转换成fileStates
	if err = LoadStateFile(stateFile); err != nil {
		return err
	}

	// 3. 遍历directory下的所有文件，如果文件名不在fileStates中，则添加到fileStates中，并更新stateFile
	if fileNames, err = k3.FetchDirectory(directory, -1); err != nil {
		return err
	}

	// 4.清理掉已经不存在的文件
	for fileName, _ = range GlobalFileStates {
		if inArray(fileNames, fileName) == false {
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

	fmt.Println("GlobalFileStatesFd: ", GlobalFileStatesFd)

	// 8. 监听watchDirectory下的文件变化，当文件发生变化时，读取文件内容，直到最后一个\n 结束， 最后更新fileStates和stateFile
	if _, err = InitWatcher(directory); err != nil {
		Clean()
		return err
	}

	GraceExit(stateFile)
	return nil
}

// GraceExit 保持进程常驻， 等待信号在退出
func GraceExit(stateFile string) {
	var (
		state      = -1
		signalChan = make(chan os.Signal, 1)
	)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	{
	EXIT:
		select {
		case sig, ok := <-signalChan:
			if !ok {
				// 直接退出，关闭
				break EXIT
			}
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				state = 0
				break EXIT
			case syscall.SIGHUP:
			default:
				state = -1
				break EXIT
			}
		}
	}

	// 关闭资源退出
	Clean()

	// 更新satefile文件内容
	SyncToSateFile(stateFile)

	time.Sleep(1 * time.Second)
	os.Exit(state)
}

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
}

// TODO 处理每次文件变化
func handleEvent(event fsnotify.Event) {
	log.Println("event:", event)
	if event.Op&fsnotify.Write == fsnotify.Write { // 写入
		offset, err := ReadFileByOffset(GlobalFileStatesFd[event.Name].Fd, GlobalFileStates[event.Name].Offset)
		if err != nil {
			k3.K3LogError("ReadFileByOffset error: %s", err)
		}
		UpdateFileStateOffset(event.Name, offset)
	} else if event.Op&fsnotify.Remove == fsnotify.Remove { // 删除
		// TODO 删除文件， 更新状态文件
	} else if event.Op&fsnotify.Create == fsnotify.Create { // 创建

	} else if event.Op&fsnotify.Rename == fsnotify.Rename { // 重命名
		log.Println("Rename event:", event.Name)
	}
}

// ReadFileByOffset 读取文件内容， 从offset开始，直到遇到\n, 返回读取后，最后的偏移量
func ReadFileByOffset(fd *os.File, offset int64) (int64, error) {
	var (
		err       error
		reader    *bufio.Reader
		content   string
		newOffset int64
	)
	newOffset = offset

	// 移动到offset位置, 从文件开始移动
	if _, err = fd.Seek(offset, io.SeekStart); err != nil {
		return newOffset, fmt.Errorf("seek file error: %s", err)
	}

	reader = bufio.NewReader(fd)
	for {
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
	k3.K3LogInfo("ReadFileByOffset: %s, content: %s", fd.Name(), content)
	return newOffset, nil
}

func UpdateFileStateOffset(fileName string, offset int64) {
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

// SyncToSateFile 将fileStates同步到stateFile, 先清空在写入, 覆盖
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

func inArray(slice []string, item string) bool {

	for _, v := range slice {
		if v == item {
			return true
		}
	}

	return false
}
