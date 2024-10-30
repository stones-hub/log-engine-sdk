package watch

import (
	"bufio"
	"context"
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

type FileState struct {
	Path          string
	Offset        int64
	StartReadTime int64
	LastReadTime  int64
	IndexName     string
}

func (f *FileState) String() string {
	return fmt.Sprintf("Path:%s, Offset:%d, StartReadTime:%v, LastReadTime:%v, IndexName:%s",
		f.Path, f.Offset, f.StartReadTime, f.LastReadTime, f.IndexName)
}

/*
  max_read_count : 100 # 监控到文件变化时，一次读取文件最大次数, 默认100次
  sync_interval : 60 # 单位秒，默认60, 程序运行过程中，要定时落盘
  state_file_path : "/state/core.json" # 记录监控文件的offset
  obsolete_interval : 1 # 单位小时, 默认1 表示定时多久时间检查文件是否已经读完了
  obsolete_date : 1 # 单位填， 默认1， 表示文件如果1天没有写入, 就查看下是不是读取完了，没读完就读完整个文件.
  obsolete_max_read_count : 1000 # 对于长时间没有读写的文件， 一次最大读取次数
*/

var (
	DefaultMaxReadCount = 200
	DefaultSyncInterval = 60 // 单位秒, 用于解决运行时，不断落盘

	DefaultObsoleteInterval     = 1    // 单位小时, 默认1 表示定时多久时间检查文件是否已经读完了
	DefaultObsoleteDate         = 1    // 单位填， 默认1， 表示文件如果1天没有写入, 就查看下是不是读取完了，没读完就读完整个文件.
	DefaultObsoleteMaxReadCount = 5000 // 对于长时间没有读写的文件， 一次最大读取次数

	GlobalFileStateFds       = make(map[string]*os.File)   // 对应所有要监控的文件fd
	GlobalFileStates         = make(map[string]*FileState) // 对应监控的所有文件的状态，映射 core.json文件
	GlobalWatchContextCancel context.CancelFunc
	GlobalWatchContext       context.Context // 控制协程主动退出
	GlobalWatchWG            *sync.WaitGroup
	GlobalWatchMutex         sync.Mutex // 控制全局变量的并发
	FileStateLock            sync.Mutex
	GlobalDataAnalytics      k3.DataAnalytics
)

func InitConsumerBatchLog() error {
	var (
		elk      *sender.ElasticSearchClient
		err      error
		consumer protocol.K3Consumer
	)
	if elk, err = sender.NewElasticsearch(config.GlobalConfig.ELK.Address,
		config.GlobalConfig.ELK.Username,
		config.GlobalConfig.ELK.Password); err != nil {
		return err
	}

	if consumer, err = k3.NewBatchConsumerWithConfig(k3.K3BatchConsumerConfig{
		Sender:        elk,
		BatchSize:     config.GlobalConfig.Consumer.ConsumerBatchSize,
		AutoFlush:     config.GlobalConfig.Consumer.ConsumerBatchAutoFlush,
		Interval:      config.GlobalConfig.Consumer.ConsumerBatchInterval,
		CacheCapacity: config.GlobalConfig.Consumer.ConsumerBatchCapacity,
	}); err != nil {
		return err
	}
	GlobalDataAnalytics = k3.NewDataAnalytics(consumer)

	return nil
}

func Run() error {

	var (
		diskPaths     = make(map[string][]string)
		diskFilePaths = make(map[string][]string)
		err           error
	)

	// 用于测试用
	config.GlobalConfig.Watch = config.Watch{
		ReadPath: map[string][]string{
			"index_nginx": []string{
				"/Users/yelei/data/code/go-projects/logs/nginx",
			},
			"index_admin": []string{
				"/Users/yelei/data/code/go-projects/logs/admin",
			},
			"index_api": []string{
				"/Users/yelei/data/code/go-projects/logs/api",
			},
		},
		StateFilePath:    "state/core.json",
		MaxReadCount:     1000,
		ObsoleteInterval: 1,
	}

	if err = InitConsumerBatchLog(); err != nil {
		k3.K3LogError("WatchRun InitConsumerBatchLog error: %s", err.Error)
		return err
	}

	// 如果state file文件没有就创建，如果有就load文件内容到stateFile
	if GlobalFileStates, err = CreateORLoadFileState(config.GlobalConfig.Watch.StateFilePath); err != nil {
		k3.K3LogError("WatchRun CreateAndLoadFileState error: %s", err.Error())
		return err
	}

	// 遍历所有的目录,找到所有需要监控的目录(包含子目录) 和 所有文件
	for indexName, paths := range config.GlobalConfig.Watch.ReadPath {
		for _, path := range paths {
			subPaths, err := FetchWatchPath(path)
			if err != nil {
				k3.K3LogError("FetchWatchPath error: %s", err.Error())
				return err
			}
			// 将所有的indexName对应的子目录全部遍历出来，封装到diskPaths中
			diskPaths[indexName] = subPaths

			filePaths, err := FetchWatchPathFile(path)
			if err != nil {
				k3.K3LogError("FetchWatchPathFile error: %s", err.Error())
				return err
			}
			// 将所有的indexName对应的文件全部遍历出来，封装到diskFilePaths中
			diskFilePaths[indexName] = filePaths
		}
	}

	/*
		watch.yaml 配置文件信息
			  read_path : # read_path每个Key的目录不可以重复，且value不可以包含相同的子集
			    index_nginx: ["/Users/yelei/data/code/go-projects/logs/nginx"] # 必须是目录
			    index_admin : [ "/Users/yelei/data/code/go-projects/logs/admin"]
			    index_api : [ "/Users/yelei/data/code/go-projects/logs/api"]
			  max_read_count : 100 # 监控到文件变化时，一次读取文件最大次数, 默认100次
			  sync_interval : 60 # 单位秒，默认60, 程序运行过程中，要定时落盘
			  state_file_path : "/state/core.json" # 记录监控文件的offset
			  obsolete_interval : 1 # 单位小时, 默认1 表示定时多久时间检查文件是否已经读完了
			  obsolete_date : 1 # 单位填， 默认1， 表示文件如果1天没有写入, 就查看下是不是读取完了，没读完就读完整个文件.
	*/
	if err = SyncFileStates2Disk(diskFilePaths, config.GlobalConfig.Watch.StateFilePath); err != nil {
		return err
	}

	// 初始化待监控的所有文件的FD
	if err = InitFileStateFds(); err != nil {
		return err
	}

	fmt.Println("-----------------------------")
	fmt.Println(GlobalFileStateFds, GlobalFileStates)
	fmt.Println("-----------------------------")

	// 开始监控, 注意多协程处理，每个index name一个线程
	GlobalWatchContext, GlobalWatchContextCancel = context.WithCancel(context.Background())
	GlobalWatchWG = &sync.WaitGroup{}
	InitWatcher(diskPaths)

	ClockSyncFileState()
	ClockCheckFileStateAndReadFile()

	return nil
}

// InitWatcher 初始化监控的协程, 每个indexName开一个协程来处理监控目录问题, 每个indexName 都对应了一批目录（是主目录遍历出来的所有子目录的集合）
func InitWatcher(diskPaths map[string][]string) {

	// 循环开协程，每个index name 一个协程
	for index, paths := range diskPaths {
		GlobalWatchWG.Add(1)
		go doWatch(index, paths)
	}

}

func doWatch(index string, paths []string) {
	var (
		childWG = &sync.WaitGroup{}
		err     error
		watcher *fsnotify.Watcher
	)

	// 协程退出
	defer GlobalWatchWG.Done()

	// 初始化协程watcher
	if watcher, err = fsnotify.NewWatcher(); err != nil {
		GlobalWatchMutex.Lock()
		GlobalWatchContextCancel()
		GlobalWatchMutex.Unlock()
		k3.K3LogError("Failed to create watcher for %s: %v\n", index, err)
		return
	}
	defer watcher.Close()

	// 开始监听目录, 如果错误就退出
	for _, path := range paths {
		if err = watcher.Add(path); err != nil {
			GlobalWatchMutex.Lock()
			GlobalWatchContextCancel()
			GlobalWatchMutex.Unlock()
			k3.K3LogError("Failed to add %s to watcher for %s: %v\n", path, index, err)
			return
		}
	}

	childWG.Add(1)
	go func() {
		defer childWG.Done()
		defer func() {
			if r := recover(); r != nil {
				GlobalWatchMutex.Lock()
				GlobalWatchContextCancel()
				GlobalWatchMutex.Unlock()
				k3.K3LogError("doWatch child goroutine panic: %s\n", r)
			}
		}()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok { // 退出子协程
					return
				}

				// TODO  检查是不是每个watcher是对应index_name创建的，如何解决每个watcher监控后文件的处理
				// TODO 检查在创建watcher之前，filestates中是不是其实已经分好了index_name对应的文件，并检查下是什么时候初始化的
				handlerEvent(watcher, event)

			case err, ok := <-watcher.Errors:
				if !ok { // 退出子协程
					return
				}
				k3.K3LogError("Child Goroutine Error reading %s: %v\n", index, err)

			case <-GlobalWatchContext.Done(): // 退出子协程
				k3.K3LogInfo("Received exit signal, child goroutine stopping....\n")
				return
			}
		}
	}()
	// 等待子协程退出
	childWG.Wait()
}

// handlerEvent 处理监控到文件的变化
func handlerEvent(watcher *fsnotify.Watcher, event fsnotify.Event) {
	if event.Op&fsnotify.Write == fsnotify.Write { // 文件写入

		// 判断map中是否存在，不存在就返回
		if _, ok := GlobalFileStates[event.Name]; !ok {
			k3.K3LogError("handlerEvent file not found: %s", event.Name)
			return
		}

		if _, ok := GlobalFileStateFds[event.Name]; !ok {
			k3.K3LogError("handlerEvent file fd not found: %s", event.Name)
			return
		}

		ReadFileByOffset(GlobalFileStateFds[event.Name], GlobalFileStates[event.Name])

	} else if event.Op&fsnotify.Create == fsnotify.Create { // 文件或目录创建

		// TODO 判断是否是目录，如果是目录需要添加监控, 如果是文件，就需要将文件添加到FileState并新增FileState fds

	} else if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename { // 文件删除或改名

	}
}

func DoRemoveAndRenameEvent() {
	// TODO 关闭文件，删除文件，更新filestate

}

func ReadFileByOffset(fd *os.File, fileState *FileState) {
	var (
		maxReadCount     = config.GlobalConfig.Watch.MaxReadCount
		currentReadCount int
		currentOffset    int64
		read             *bufio.Reader
		content          string
	)

	if maxReadCount < 0 || maxReadCount > DefaultMaxReadCount {
		maxReadCount = DefaultMaxReadCount
	}

	currentReadCount = 0
	currentOffset = fileState.Offset

	for currentReadCount < maxReadCount {
		currentReadCount++

		if _, err := fd.Seek(currentOffset, 0); err != nil {
			k3.K3LogError("ReadFileByOffset Seek error: %s", err.Error())
			return
		}

		read = bufio.NewReader(fd)
		line, err := read.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				k3.K3LogInfo("ReadFileByOffset ReadString error: %s", err.Error())
			} else {
				k3.K3LogError("ReadFileByOffset ReadString error: %s", err.Error())
			}
			break
		}

		if len(line) > 0 {
			content += line
		}
	}

	if len(content) > 0 {
		if err := SendData2Consumer(content, fileState); err != nil {
			k3.K3LogError("SendData2Consumer error: %s", err.Error())
		}

		currentOffset += int64(len(content))
	}

	if err := SyncFileState(fileState.Path, currentOffset); err != nil {
		k3.K3LogError("SyncFileState: %s\n", err)
		return
	}
}

func Close() {
	// 关闭所有打开的文件
	for _, fd := range GlobalFileStateFds {
		fd.Close()
	}
	GlobalWatchContextCancel()
	GlobalWatchWG.Wait()
	GlobalDataAnalytics.Close()
}

func InitFileStateFds() error {
	var (
		err error
	)

	for filePath, _ := range GlobalFileStates {
		if GlobalFileStateFds[filePath], err = os.OpenFile(filePath, os.O_RDONLY, 0666); err != nil {
			return fmt.Errorf("InitFileStateFds open file error: %s", err.Error())
		}

		if GlobalFileStates[filePath].StartReadTime == 0 {
			GlobalFileStates[filePath].StartReadTime = time.Now().Unix()
		}
	}

	return nil
}

// SyncFileStates2Disk 将FileState数据写入到磁盘, 先删除在覆盖
func SyncFileStates2Disk(diskFilePaths map[string][]string, filePath string) error {
	var (
		fd      *os.File
		err     error
		encoder *json.Encoder
	)

	SyncWatchFiles2FileStates(diskFilePaths)
	SyncFileStates2WatchFiles(diskFilePaths)

	FileStateLock.Lock()
	defer FileStateLock.Unlock()

	// 将数据写入到 state_file_path
	if fd, err = os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666); err != nil {
		return fmt.Errorf("open state file error: %s", err.Error())
	}

	defer fd.Close()

	encoder = json.NewEncoder(fd)

	if err = encoder.Encode(GlobalFileStates); err != nil {
		return fmt.Errorf("encode state file error: %s", err.Error())
	}

	return nil
}

// SyncWatchFiles2FileStates 初始化时
// 遍历硬盘上被监控目录的所有文件, 判断文件是否在FileState中，如果不在，证明是新增的文件, 则添加到FileState中
func SyncWatchFiles2FileStates(watchFiles map[string][]string) {
	for index, files := range watchFiles {
		for _, diskFilePath := range files {
			if !CheckDiskFileIsExistInFileStates(diskFilePath) {
				GlobalFileStates[diskFilePath] = &FileState{
					Path:      diskFilePath,
					Offset:    0,
					IndexName: index,
				}
			}
		}
	}
}

// CheckDiskFileIsExistInFileStates 判断文件是否在FileState中
func CheckDiskFileIsExistInFileStates(diskFilePath string) bool {
	for filePath := range GlobalFileStates {
		if filePath == diskFilePath {
			return true
		}
	}
	return false
}

// SyncFileStates2WatchFiles 初始化时
// 遍历FileState中记录的所有文件，如果文件不存在于本地硬盘中，证明已经被删除了，对应在FileState中删除
func SyncFileStates2WatchFiles(watchFiles map[string][]string) {
	for fileStatePath := range GlobalFileStates {
		if !CheckFileStateIsExistInDiskFiles(fileStatePath, watchFiles) {
			delete(GlobalFileStates, fileStatePath)
		}
	}
}

// CheckFileStateIsExistInDiskFiles 判断FileState是否在硬盘中
func CheckFileStateIsExistInDiskFiles(fileStatePath string, watchFiles map[string][]string) bool {
	for _, files := range watchFiles {
		for _, diskFilePath := range files {
			if diskFilePath == fileStatePath {
				return true
			}
		}
	}
	return false
}

// CreateORLoadFileState 创建并加载状态文件
func CreateORLoadFileState(fileSatePath string) (map[string]*FileState, error) {
	var (
		fd         *os.File
		err        error
		decoder    *json.Decoder
		fileStates = make(map[string]*FileState)
	)
	// 判断文件是否存在, 不存在就创建, 存在就将文本内容加载出来,映射到SateFile中
	if fd, err = os.OpenFile(fileSatePath, os.O_CREATE|os.O_RDWR, 0666); err != nil {
		return nil, err
	}
	defer fd.Close()

	decoder = json.NewDecoder(fd)

	if err = decoder.Decode(&fileStates); err != nil && err != io.EOF {
		return nil, err
	}

	return fileStates, nil
}

// FetchWatchPath 获取需要监控的目录中的所有子目录
func FetchWatchPath(watchPath string) ([]string, error) {

	var (
		paths []string
		err   error
	)

	if err = filepath.WalkDir(watchPath, func(currentPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			paths = append(paths, currentPath)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return paths, err
}

// FetchWatchPathFile 获取监控目录中的所有文件
func FetchWatchPathFile(watchPath string) ([]string, error) {
	return k3.FetchDirectory(watchPath, -1)
}

// ForceSyncFileState 强制同步FileState, 先清空，在同步
func ForceSyncFileState() error {
	var (
		fd      *os.File
		err     error
		encoder *json.Encoder
	)
	FileStateLock.Lock()
	defer FileStateLock.Unlock()
	if fd, err = os.OpenFile(config.GlobalConfig.Watch.StateFilePath, os.O_RDWR|os.O_TRUNC, 0666); err != nil {
		return fmt.Errorf("open state file error: %s", err.Error())
	}

	defer fd.Close()
	encoder = json.NewEncoder(fd)

	if err = encoder.Encode(&GlobalFileStates); err != nil {
		return fmt.Errorf("encode state file error: %s", err.Error())
	}
	return nil
}

// ClockSyncFileState 定时将内存中的file state 写入硬盘
func ClockSyncFileState() {
	var (
		syncInterval = config.GlobalConfig.Watch.ObsoleteInterval
		t            *time.Ticker
	)

	// 获取监控时间间隔
	if syncInterval <= 0 || syncInterval > DefaultSyncInterval {
		syncInterval = DefaultSyncInterval
	}
	// 创建定时器
	t = time.NewTicker(time.Duration(syncInterval) * time.Second)

	GlobalWatchWG.Add(1)
	go func() {
		defer GlobalWatchWG.Done()
		defer func() {
			t.Stop()
		}()
		for {
			select {
			case <-GlobalWatchContext.Done():
				return
				// 获取到定时时间
			case <-t.C:
				// 执行同步逻辑任务
				if err := ForceSyncFileState(); err != nil {
					k3.K3LogError("ForceSyncFileState: %s\n", err)
				}
			}
		}
	}()
}

// ClockCheckFileStateAndReadFile 定时监控file states 中的文件状态，是不是长时间没有写入，如果没有就一次性读取完, 注意子协程的回收处理
func ClockCheckFileStateAndReadFile() {

	var (
		obInterval = config.GlobalConfig.Watch.ObsoleteInterval
		t          *time.Ticker
	)
	if obInterval <= 0 || obInterval > DefaultObsoleteInterval {
		obInterval = DefaultObsoleteInterval
	}
	// 创建定时器, 如果定时器间隔时间小于，逻辑处理时间，信号会丢弃，因为定时器的实现原理是chan size = 1
	t = time.NewTicker(time.Duration(obInterval) * time.Hour)

	GlobalWatchWG.Add(1)
	go func() {
		defer GlobalWatchWG.Done()
		defer func() {
			t.Stop()
		}()

		for {
			select {
			case <-GlobalWatchContext.Done():
				return
			case <-t.C:
				HandleReadFileAndSendData()
			}
		}
	}()
}

// HandleReadFileAndSendData 处理读取文件并发送数据
func HandleReadFileAndSendData() {
	var (
		wg           = &sync.WaitGroup{}
		obsoleteDate = config.GlobalConfig.Watch.ObsoleteDate
	)

	if obsoleteDate < 0 || obsoleteDate > DefaultObsoleteDate {
		obsoleteDate = DefaultObsoleteDate
	}

	for fileName, fileState := range GlobalFileStates {
		now := time.Now()
		lastReadTime := time.Unix(fileState.LastReadTime, 0)
		// 判断当前时间点是否超过N天未读写
		if now.Sub(lastReadTime) > 24*time.Hour*time.Duration(obsoleteDate) {

			// 判断文件是否读取完，如果没有，就一次性全部读取, 但要控制内存, 放内存数据太大
			if fd, ok := GlobalFileStateFds[fileName]; ok && fd != nil {
				if fstat, err := fd.Stat(); err != nil {
					k3.K3LogError("stat file error: %s", err)
					continue
				} else {
					if fstat.Size() != fileState.Offset {
						// 开协程，一次性读取所有内容, 但最好有个阀值， 避免内存过多
						wg.Add(1)
						go ReadFileAndSyncFileState(fd, fileState, wg)
					}
				}
			}
		}
	}

	wg.Wait()
}

func ReadFileAndSyncFileState(fd *os.File, fileState *FileState, wg *sync.WaitGroup) {
	var (
		obsoleteMaxReadCount = config.GlobalConfig.Watch.ObsoleteMaxReadCount
		reader               *bufio.Reader
		lastOffset           int64
		currentReadCount     int
		err                  error
		content              string
	)

	if obsoleteMaxReadCount < 0 || obsoleteMaxReadCount > DefaultObsoleteMaxReadCount {
		obsoleteMaxReadCount = DefaultObsoleteMaxReadCount
	}

	defer wg.Done()

	if _, err = fd.Seek(fileState.Offset, io.SeekStart); err != nil {
		k3.K3LogError("seek file error: %s", err)
		return
	}

	lastOffset = fileState.Offset
	currentReadCount = 0
	reader = bufio.NewReader(fd)

	for currentReadCount < obsoleteMaxReadCount {
		currentReadCount++
		line, err := reader.ReadString('\n')

		// 读取文件错误, 有可能读完了
		if err != nil {
			if err == io.EOF {
				k3.K3LogInfo("read file eof\n")
			} else {
				k3.K3LogError("read file error: %s\n", err)
			}
			break
		}

		if len(line) > 0 {
			content += line
		}
	}

	if len(content) > 0 {
		if err = SendData2Consumer(content, fileState); err != nil {
			k3.K3LogError("SendData2Consumer: %s\n", err)
			return
		}

		// 发送成功以后，才可以改变偏移量
		lastOffset += int64(len(content))
	}

	if err = SyncFileState(fileState.Path, lastOffset); err != nil {
		k3.K3LogError("SyncFileState: %s\n", err)
		return
	}
}

// SendData2Consumer 未使用 将数据发送给consumer
func SendData2Consumer(content string, fileState *FileState) error {

	var (
		ip    string
		ips   []string
		datas []string
		err   error
	)

	if ips, err = k3.GetLocalIPs(); err != nil {
		return fmt.Errorf("get local ips error: %s", err)
	} else {
		ip = ips[0]
	}

	datas = strings.Split(content, "\n")
	for _, data := range datas {
		data = strings.TrimSpace(data)
		data = strings.Trim(data, "\n")
		if len(data) == 0 {
			continue
		}

		if err = GlobalDataAnalytics.Track(config.GlobalConfig.Account.AccountId, config.GlobalConfig.Account.AppId, ip, fileState.IndexName,
			map[string]interface{}{
				"_data": data,
			}); err != nil {
			k3.K3LogError("Track: %s\n", err.Error())
		}

	}

	return nil
}

// SyncFileState 更新file state
func SyncFileState(filePath string, lastOffset int64) error {
	var (
		fd      *os.File
		err     error
		encoder *json.Encoder
	)

	FileStateLock.Lock()
	defer FileStateLock.Unlock()

	if GlobalFileStates[filePath].StartReadTime == 0 {
		GlobalFileStates[filePath].StartReadTime = time.Now().Unix()
	}
	GlobalFileStates[filePath].Offset = lastOffset              // 更新最后偏移量
	GlobalFileStates[filePath].LastReadTime = time.Now().Unix() // 更新最后读取时间

	// 写入文件

	if fd, err = os.OpenFile(config.GlobalConfig.Watch.StateFilePath, os.O_RDWR|os.O_TRUNC, 0666); err != nil {
		return fmt.Errorf("open state file error: %s", err.Error())
	}

	defer fd.Close()
	encoder = json.NewEncoder(fd)

	if err = encoder.Encode(&GlobalFileStates); err != nil {
		return fmt.Errorf("encode state file error: %s", err.Error())
	}
	return nil
}
