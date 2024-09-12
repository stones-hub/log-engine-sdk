package file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

const stateFile = "state.txt" // 存储已处理文件的状态文件

// loadState 加载状态文件中的内容
func loadState() []string {
	state, err := os.ReadFile(stateFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
	return strings.Split(string(state), "\n")
}

// saveState 保存当前状态到文件
func saveState(files []string) error {
	data := strings.Join(files, "\n")
	return os.WriteFile(stateFile, []byte(data), 0644)
}

// handleEvents 处理文件系统事件
func handleEvents(watcher *fsnotify.Watcher, processedFiles map[string]bool) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if _, exists := processedFiles[event.Name]; !exists {
					fmt.Printf("Processing file: %s\n", event.Name)
					// 这里可以添加具体的文件处理逻辑
					processedFiles[event.Name] = true
					saveState(getProcessedFiles(processedFiles))
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

// scanDirectory 深度递归遍历目录并将所有文件记录下来
func scanDirectory(dir string, maxDepth int, depth int, processedFiles map[string]bool) {
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if maxDepth >= 0 && depth > maxDepth {
				return filepath.SkipDir
			}
			return nil // 继续递归
		}
		if _, exists := processedFiles[path]; !exists {
			fmt.Printf("Initial processing: %s\n", path)
			processedFiles[path] = true
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to walk directory: %v", err)
	}

	saveState(getProcessedFiles(processedFiles))
}

func main() {
	// 初始化状态
	processedFiles := make(map[string]bool)
	for _, file := range loadState() {
		if file != "" {
			processedFiles[file] = true
		}
	}

	// 设置文件系统监听
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	dirToWatch := "./target_directory" // 要监视的目录
	maxDepth := 3                      // 设置最大深度，-1 表示无限深度

	err = watcher.Add(dirToWatch)
	if err != nil {
		log.Fatal(err)
	}

	// 初始扫描目录
	scanDirectory(dirToWatch, maxDepth, 0, processedFiles)

	// 开始监听文件系统事件
	handleEvents(watcher, processedFiles)
}

// getProcessedFiles 获取已处理的文件列表
func getProcessedFiles(files map[string]bool) []string {
	var list []string
	for file := range files {
		list = append(list, file)
	}
	return list
}
