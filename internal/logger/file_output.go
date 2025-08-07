package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileOutput 文件输出器
type FileOutput struct {
	config      OutputConfig
	currentFile *os.File
	currentDate string
	mutex       sync.RWMutex
	closed      bool
}

// NewFileOutput 创建文件输出器
func NewFileOutput(config OutputConfig) (*FileOutput, error) {
	output := &FileOutput{
		config: config,
	}

	// 确保目录存在
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 初始化当前文件
	if err := output.rotateFile(); err != nil {
		return nil, fmt.Errorf("初始化日志文件失败: %w", err)
	}

	// 启动清理任务
	go output.cleanupTask()

	return output, nil
}

// Write 写入日志数据
func (f *FileOutput) Write(data []byte) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.closed {
		return fmt.Errorf("文件输出器已关闭")
	}

	// 检查是否需要轮转文件
	if f.needRotate() {
		if err := f.rotateFile(); err != nil {
			return fmt.Errorf("轮转日志文件失败: %w", err)
		}
	}

	// 写入数据
	if f.currentFile != nil {
		_, err := f.currentFile.Write(data)
		if err != nil {
			return fmt.Errorf("写入日志文件失败: %w", err)
		}

		// 立即刷新到磁盘
		if err := f.currentFile.Sync(); err != nil {
			return fmt.Errorf("刷新日志文件失败: %w", err)
		}
	}

	return nil
}

// Close 关闭输出器
func (f *FileOutput) Close() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.closed = true

	if f.currentFile != nil {
		err := f.currentFile.Close()
		f.currentFile = nil
		return err
	}

	return nil
}

// needRotate 检查是否需要轮转文件
func (f *FileOutput) needRotate() bool {
	now := time.Now()
	var currentPeriod string

	switch f.config.Period {
	case PeriodHour:
		currentPeriod = now.Format("2006010215")
	case PeriodDay:
		currentPeriod = now.Format("20060102")
	default:
		currentPeriod = now.Format("20060102")
	}

	return f.currentDate != currentPeriod
}

// rotateFile 轮转文件
func (f *FileOutput) rotateFile() error {
	now := time.Now()
	var newDate string

	switch f.config.Period {
	case PeriodHour:
		newDate = now.Format("2006010215")
	case PeriodDay:
		newDate = now.Format("20060102")
	default:
		newDate = now.Format("20060102")
	}
	fileName := strings.TrimSuffix(f.config.File, ".log")
	// 如果有当前文件，先关闭并重命名
	if f.currentFile != nil {
		f.currentFile.Close()

		// 重命名旧文件
		oldPath := filepath.Join(f.config.Dir, fileName+".log")
		newPath := filepath.Join(f.config.Dir, fmt.Sprintf("%s-%s.log", fileName, f.currentDate))

		// 只有当文件存在且不是当前周期时才重命名
		if _, err := os.Stat(oldPath); err == nil && f.currentDate != "" && f.currentDate != newDate {
			if err := os.Rename(oldPath, newPath); err != nil {
				// 重命名失败不应该阻止创建新文件
				fmt.Printf("重命名日志文件失败: %v\n", err)
			}
		}
	}

	// 创建新文件
	filePath := filepath.Join(f.config.Dir, fileName+".log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %w", err)
	}

	f.currentFile = file
	f.currentDate = newDate

	return nil
}

// cleanupTask 清理过期文件的任务
func (f *FileOutput) cleanupTask() {
	ticker := time.NewTicker(time.Hour) // 每小时检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.cleanupExpiredFiles()
		}

		// 检查是否已关闭
		f.mutex.RLock()
		closed := f.closed
		f.mutex.RUnlock()

		if closed {
			return
		}
	}
}

// cleanupExpiredFiles 清理过期文件
func (f *FileOutput) cleanupExpiredFiles() {
	if f.config.Expire <= 0 {
		return // 不清理
	}

	expireTime := time.Now().AddDate(0, 0, -f.config.Expire)

	// 读取目录中的所有文件
	files, err := os.ReadDir(f.config.Dir)
	if err != nil {
		fmt.Printf("读取日志目录失败: %v\n", err)
		return
	}

	// 查找需要删除的文件
	var filesToDelete []string
	prefix := f.config.File + "-"
	suffix := ".log"

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}

		// 获取文件信息
		info, err := file.Info()
		if err != nil {
			continue
		}

		// 检查文件是否过期
		if info.ModTime().Before(expireTime) {
			filesToDelete = append(filesToDelete, filepath.Join(f.config.Dir, name))
		}
	}

	// 删除过期文件
	for _, filePath := range filesToDelete {
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("删除过期日志文件失败 %s: %v\n", filePath, err)
		} else {
			fmt.Printf("删除过期日志文件: %s\n", filePath)
		}
	}
}

// GetLogFiles 获取日志文件列表
func (f *FileOutput) GetLogFiles() ([]LogFileInfo, error) {
	files, err := os.ReadDir(f.config.Dir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %w", err)
	}

	var logFiles []LogFileInfo
	prefix := f.config.File
	suffix := ".log"

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		logFile := LogFileInfo{
			Name:      name,
			Path:      filepath.Join(f.config.Dir, name),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsCurrent: name == f.config.File+".log",
		}

		logFiles = append(logFiles, logFile)
	}

	// 按修改时间排序
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].ModTime.After(logFiles[j].ModTime)
	})

	return logFiles, nil
}

// ReadLogFile 读取日志文件内容
func (f *FileOutput) ReadLogFile(filename string, offset int64, limit int64) ([]byte, error) {
	filePath := filepath.Join(f.config.Dir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("日志文件不存在: %s", filename)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	// 移动到指定偏移量
	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("移动文件指针失败: %w", err)
		}
	}

	// 读取指定长度的数据
	if limit <= 0 {
		limit = 1024 * 1024 // 默认1MB
	}

	buffer := make([]byte, limit)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	return buffer[:n], nil
}

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	IsCurrent bool      `json:"is_current"`
}
