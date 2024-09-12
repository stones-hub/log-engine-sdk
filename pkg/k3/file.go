package k3

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// ProgressRecord 结构体用于存储文件及其读取进度
type ProgressRecord struct {
	FileName string
	Position int64
}

// LoadProgress 从进度文件加载进度记录
func LoadProgress(progressFile string) (map[string]int64, error) {
	file, err := os.Open(progressFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	progressMap := make(map[string]int64)

	for scanner.Scan() {
		line := scanner.Text()
		parts := filepath.SplitList(line)
		if len(parts) == 2 {
			position, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				continue
			}
			progressMap[parts[0]] = position
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return progressMap, nil
}

// SaveProgress 将进度记录保存到进度文件
func SaveProgress(progressFile string, progressMap map[string]int64) error {
	file, err := os.Create(progressFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for fileName, position := range progressMap {
		_, err := fmt.Fprintf(file, "%s %d\n", fileName, position)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReadFile 读取文件内容并记录进度
func ReadFile(fileName string, progressMap map[string]int64) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var line []byte
	var isPrefix bool

	// 从上次读取的位置继续读取
	if position, ok := progressMap[fileName]; ok {
		_, err := file.Seek(position, io.SeekStart)
		if err != nil {
			return err
		}
	}

	for {
		line, isPrefix, err = reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		fmt.Println(string(line))

		// 更新进度
		if isPrefix {
			continue
		}

		position, _ := file.Seek(0, io.SeekCurrent)
		progressMap[fileName] = position
	}

	return nil
}

func do() {
	dirPath := "/path/to/directory"
	progressFile := "progress.txt"

	// 加载进度记录
	progressMap, err := LoadProgress(progressFile)
	if err != nil {
		log.Fatalf("Error loading progress: %v", err)
	}

	// 遍历目录
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			err := ReadFile(path, progressMap)
			if err != nil {
				log.Printf("Error reading file %s: %v", path, err)
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	// 保存进度记录
	err = SaveProgress(progressFile, progressMap)
	if err != nil {
		log.Fatalf("Error saving progress: %v", err)
	}
}
