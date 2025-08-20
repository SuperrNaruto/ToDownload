package progress

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
)

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	mu       sync.RWMutex
	tasks    map[string]*TaskProgress
	ctx      *ext.Context
	logger   *log.Logger
	updates  chan *ProgressUpdate
	stopCh   chan struct{}
	running  bool
}

// TaskProgress 任务进度
type TaskProgress struct {
	TaskID          string        `json:"task_id"`
	ChatID          int64         `json:"chat_id"`
	MessageID       int           `json:"message_id"`
	FileName        string        `json:"file_name"`
	Status          TaskStatus    `json:"status"`
	Progress        int           `json:"progress"`        // 0-100
	TotalBytes      int64         `json:"total_bytes"`
	ProcessedBytes  int64         `json:"processed_bytes"`
	Speed           int64         `json:"speed"`           // bytes/second
	ETA             time.Duration `json:"eta"`
	StartTime       time.Time     `json:"start_time"`
	LastUpdate      time.Time     `json:"last_update"`
	ErrorMessage    string        `json:"error_message"`
}

// TaskStatus 任务状态
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusStarting   TaskStatus = "starting"
	StatusDownloading TaskStatus = "downloading"
	StatusUploading  TaskStatus = "uploading"
	StatusProcessing TaskStatus = "processing"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
	StatusCanceled   TaskStatus = "canceled"
)

// ProgressUpdate 进度更新
type ProgressUpdate struct {
	TaskID         string
	Progress       int
	ProcessedBytes int64
	Speed          int64
	Status         TaskStatus
	ErrorMessage   string
}

// NewProgressTracker 创建新的进度跟踪器
func NewProgressTracker(ctx *ext.Context) *ProgressTracker {
	return &ProgressTracker{
		tasks:   make(map[string]*TaskProgress),
		ctx:     ctx,
		logger:  log.FromContext(ctx),
		updates: make(chan *ProgressUpdate, 100),
		stopCh:  make(chan struct{}),
	}
}

// Start 启动进度跟踪器
func (pt *ProgressTracker) Start() {
	pt.mu.Lock()
	if pt.running {
		pt.mu.Unlock()
		return
	}
	pt.running = true
	pt.mu.Unlock()
	
	go pt.updateLoop()
}

// Stop 停止进度跟踪器
func (pt *ProgressTracker) Stop() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if !pt.running {
		return
	}
	
	close(pt.stopCh)
	pt.running = false
}

// StartTask 开始跟踪任务
func (pt *ProgressTracker) StartTask(taskID, fileName string, chatID int64, totalBytes int64) (*TaskProgress, error) {
	// 先发送初始消息
	template := msgelem.NewProcessingTemplate("开始处理文件", "")
	template.AddItem("📄", "文件名", fileName, msgelem.ItemTypeCode)
	template.AddItem("📏", "大小", fmt.Sprintf("%d", totalBytes), msgelem.ItemTypeSize)
	template.AddItem("📊", "状态", string(StatusStarting), msgelem.ItemTypeStatus)
	template.AddItem("⏱️", "进度", "0%", msgelem.ItemTypeText)
	
	msg, err := pt.ctx.SendMessage(chatID, ext.SendMessageString(template.BuildMessage()))
	if err != nil {
		return nil, fmt.Errorf("failed to send progress message: %w", err)
	}
	
	messageID := msg.(*tg.Updates).Updates[0].(*tg.UpdateNewMessage).Message.(*tg.Message).ID
	
	taskProgress := &TaskProgress{
		TaskID:         taskID,
		ChatID:         chatID,
		MessageID:      messageID,
		FileName:       fileName,
		Status:         StatusStarting,
		Progress:       0,
		TotalBytes:     totalBytes,
		ProcessedBytes: 0,
		StartTime:      time.Now(),
		LastUpdate:     time.Now(),
	}
	
	pt.mu.Lock()
	pt.tasks[taskID] = taskProgress
	pt.mu.Unlock()
	
	return taskProgress, nil
}

// UpdateProgress 更新任务进度
func (pt *ProgressTracker) UpdateProgress(taskID string, processedBytes int64, status TaskStatus) {
	update := &ProgressUpdate{
		TaskID:         taskID,
		ProcessedBytes: processedBytes,
		Status:         status,
	}
	
	pt.mu.RLock()
	task := pt.tasks[taskID]
	pt.mu.RUnlock()
	
	if task != nil && task.TotalBytes > 0 {
		update.Progress = int((processedBytes * 100) / task.TotalBytes)
		
		// 计算速度
		elapsed := time.Since(task.StartTime).Seconds()
		if elapsed > 0 {
			update.Speed = int64(float64(processedBytes) / elapsed)
		}
	}
	
	select {
	case pt.updates <- update:
	default:
		// 缓冲区满了，跳过这次更新
	}
}

// CompleteTask 完成任务
func (pt *ProgressTracker) CompleteTask(taskID string, success bool, errorMessage string) {
	status := StatusCompleted
	if !success {
		status = StatusFailed
	}
	
	update := &ProgressUpdate{
		TaskID:       taskID,
		Status:       status,
		Progress:     100,
		ErrorMessage: errorMessage,
	}
	
	select {
	case pt.updates <- update:
	default:
	}
	
	// 延迟删除任务记录
	go func() {
		time.Sleep(30 * time.Second)
		pt.mu.Lock()
		delete(pt.tasks, taskID)
		pt.mu.Unlock()
	}()
}

// updateLoop 更新循环
func (pt *ProgressTracker) updateLoop() {
	ticker := time.NewTicker(2 * time.Second) // 每2秒更新一次界面
	defer ticker.Stop()
	
	for {
		select {
		case <-pt.stopCh:
			return
		case update := <-pt.updates:
			pt.handleProgressUpdate(update)
		case <-ticker.C:
			pt.refreshStaleMessages()
		}
	}
}

// handleProgressUpdate 处理进度更新
func (pt *ProgressTracker) handleProgressUpdate(update *ProgressUpdate) {
	pt.mu.Lock()
	task := pt.tasks[update.TaskID]
	if task == nil {
		pt.mu.Unlock()
		return
	}
	
	// 更新任务信息
	task.Status = update.Status
	task.Progress = update.Progress
	task.ProcessedBytes = update.ProcessedBytes
	task.Speed = update.Speed
	task.LastUpdate = time.Now()
	if update.ErrorMessage != "" {
		task.ErrorMessage = update.ErrorMessage
	}
	
	// 计算ETA
	if task.Speed > 0 && task.TotalBytes > task.ProcessedBytes {
		remainingBytes := task.TotalBytes - task.ProcessedBytes
		task.ETA = time.Duration(remainingBytes/task.Speed) * time.Second
	}
	
	// 复制任务信息以便在解锁后使用
	taskCopy := *task
	pt.mu.Unlock()
	
	// 更新消息
	pt.updateTaskMessage(&taskCopy)
}

// refreshStaleMessages 刷新过时的消息
func (pt *ProgressTracker) refreshStaleMessages() {
	pt.mu.RLock()
	var staleTasks []*TaskProgress
	for _, task := range pt.tasks {
		if time.Since(task.LastUpdate) > 5*time.Second && 
		   task.Status != StatusCompleted && 
		   task.Status != StatusFailed {
			staleTasks = append(staleTasks, &TaskProgress{
				TaskID:         task.TaskID,
				ChatID:         task.ChatID,
				MessageID:      task.MessageID,
				FileName:       task.FileName,
				Status:         task.Status,
				Progress:       task.Progress,
				TotalBytes:     task.TotalBytes,
				ProcessedBytes: task.ProcessedBytes,
				Speed:          task.Speed,
				ETA:            task.ETA,
				StartTime:      task.StartTime,
				LastUpdate:     task.LastUpdate,
				ErrorMessage:   task.ErrorMessage,
			})
		}
	}
	pt.mu.RUnlock()
	
	for _, task := range staleTasks {
		pt.updateTaskMessage(task)
	}
}

// updateTaskMessage 更新任务消息
func (pt *ProgressTracker) updateTaskMessage(task *TaskProgress) {
	var template *msgelem.MessageTemplate
	
	switch task.Status {
	case StatusCompleted:
		template = msgelem.NewSuccessTemplate("文件处理完成", "")
		template.AddItem("📄", "文件名", task.FileName, msgelem.ItemTypeCode)
		template.AddItem("📏", "大小", fmt.Sprintf("%d", task.TotalBytes), msgelem.ItemTypeSize)
		template.AddItem("⏱️", "用时", msgelem.FormatDuration(time.Since(task.StartTime)), msgelem.ItemTypeText)
		if task.Speed > 0 {
			template.AddItem("🚀", "平均速度", msgelem.FormatSize(task.Speed)+"/s", msgelem.ItemTypeText)
		}
	case StatusFailed:
		template = msgelem.NewErrorTemplate("文件处理失败", task.ErrorMessage)
		template.AddItem("📄", "文件名", task.FileName, msgelem.ItemTypeCode)
		if task.ProcessedBytes > 0 {
			template.AddItem("📊", "已处理", msgelem.FormatSize(task.ProcessedBytes), msgelem.ItemTypeText)
		}
	default:
		template = msgelem.NewProcessingTemplate("文件处理中", "")
		template.AddItem("📄", "文件名", task.FileName, msgelem.ItemTypeCode)
		template.AddItem("📏", "大小", msgelem.FormatSize(task.TotalBytes), msgelem.ItemTypeText)
		template.AddItem("📊", "进度", fmt.Sprintf("%d/%d", task.ProcessedBytes, task.TotalBytes), msgelem.ItemTypeProgress)
		template.AddItem("📈", "状态", string(task.Status), msgelem.ItemTypeStatus)
		
		if task.Speed > 0 {
			template.AddItem("🚀", "速度", msgelem.FormatSize(task.Speed)+"/s", msgelem.ItemTypeText)
		}
		
		if task.ETA > 0 {
			template.AddItem("⏱️", "预计剩余", msgelem.FormatDuration(task.ETA), msgelem.ItemTypeText)
		}
		
		elapsed := time.Since(task.StartTime)
		template.AddItem("⌚", "已用时", msgelem.FormatDuration(elapsed), msgelem.ItemTypeText)
	}
	
	// 更新消息
	_, err := pt.ctx.Bot.EditMessage(pt.ctx, &tg.MessagesEditMessageRequest{
		Peer:    &tg.PeerUser{UserID: task.ChatID},
		ID:      task.MessageID,
		Message: template.BuildMessage(),
	})
	
	if err != nil {
		pt.logger.Errorf("Failed to update progress message for task %s: %v", task.TaskID, err)
	}
}

// GetTaskProgress 获取任务进度
func (pt *ProgressTracker) GetTaskProgress(taskID string) (*TaskProgress, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	
	task, exists := pt.tasks[taskID]
	if !exists {
		return nil, false
	}
	
	// 返回副本
	taskCopy := *task
	return &taskCopy, true
}

// CancelTask 取消任务
func (pt *ProgressTracker) CancelTask(taskID string) {
	pt.UpdateProgress(taskID, 0, StatusCanceled)
}

// 全局进度跟踪器实例
var globalTracker *ProgressTracker

// InitGlobalTracker 初始化全局进度跟踪器
func InitGlobalTracker(ctx *ext.Context) {
	if globalTracker != nil {
		globalTracker.Stop()
	}
	globalTracker = NewProgressTracker(ctx)
	globalTracker.Start()
}

// GetGlobalTracker 获取全局进度跟踪器
func GetGlobalTracker() *ProgressTracker {
	return globalTracker
}

// StopGlobalTracker 停止全局进度跟踪器
func StopGlobalTracker() {
	if globalTracker != nil {
		globalTracker.Stop()
		globalTracker = nil
	}
}