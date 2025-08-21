package batchtftask

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

type Progress struct {
	MessageID         int
	ChatID            int64
	start             time.Time
	lastUpdatePercent atomic.Int32
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	p.start = time.Now()
	p.lastUpdatePercent.Store(0)
	log.FromContext(ctx).Debugf("Batch task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	
	// 使用新的模板系统，简化初始状态显示
	template := msgelem.NewInfoTemplate("🚀 开始批量下载", "")
	template.AddItem("📦", "文件数量", strconv.Itoa(info.Count()), msgelem.ItemTypeText)
	template.AddItem("📏", "总大小", msgelem.FormatSize(info.TotalSize()), msgelem.ItemTypeText)
	// 移除多余的"状态"显示，直接进入下载
	
	text, entities := template.BuildFormattedMessage()
	
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(info.TaskID()),
				},
			},
		},
	}
	
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		peer := &tg.InputPeerUser{UserID: p.ChatID}
		if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, markup); err != nil {
			log.Warn("Failed to edit message for batch task start", "error", err, "task_id", info.TaskID())
		}
		return
	}
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo) {
	if !shouldUpdateProgress(info.TotalSize(), info.Downloaded(), int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int((info.Downloaded() * 100) / info.TotalSize())
	if p.lastUpdatePercent.Load() == int32(percent) {
		return
	}
	p.lastUpdatePercent.Store(int32(percent))
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), info.Downloaded(), info.TotalSize())
	
	// 使用新的模板系统，简化进度显示
	template := msgelem.NewProcessingTemplate("批量下载中", "")
	
	// 基本信息
	template.AddItem("📦", "文件数量", strconv.Itoa(info.Count()), msgelem.ItemTypeText)
	
	// 进度信息
	template.AddProgressBar("📊", "总体进度", info.Downloaded(), info.TotalSize(), 12)
	
	// 简化的当前状态信息
	processingCount := len(info.Processing())
	if processingCount > 0 {
		statusText := fmt.Sprintf("%d 个文件", processingCount)
		template.AddItem("🔄", "正在处理", statusText, msgelem.ItemTypeText)
	}
	
	// 速度信息
	speed := dlutil.GetSpeed(info.Downloaded(), p.start)
	if speed > 0 {
		template.AddItem("🚀", "平均速度", msgelem.FormatSize(int64(speed))+"/s", msgelem.ItemTypeText)
	}
	
	text, entities := template.BuildFormattedMessage()
	
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(info.TaskID()),
				},
			},
		},
	}
	
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		peer := &tg.InputPeerUser{UserID: p.ChatID}
		if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, markup); err != nil {
			log.Warn("Failed to edit message for batch task progress", "error", err, "task_id", info.TaskID())
		}
		return
	}
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	if err != nil {
		log.FromContext(ctx).Errorf("Batch task %s failed: %s", info.TaskID(), err)
	} else {
		log.FromContext(ctx).Debugf("Batch task %s completed successfully", info.TaskID())
	}

	var template *msgelem.MessageTemplate
	
	if err != nil {
		if errors.Is(err, context.Canceled) {
			template = msgelem.NewErrorTemplate("批量任务已取消", "")
			template.AddItem("📦", "文件数量", strconv.Itoa(info.Count()), msgelem.ItemTypeText)
		} else {
			template = msgelem.NewErrorTemplate("批量下载失败", "")
			template.AddItem("📦", "文件数量", strconv.Itoa(info.Count()), msgelem.ItemTypeText)
			template.AddItem("❗", "错误信息", err.Error(), msgelem.ItemTypeText)
		}
	} else {
		template = msgelem.NewSuccessTemplate("批量下载完成", "")
		template.AddItem("📦", "文件数量", strconv.Itoa(info.Count()), msgelem.ItemTypeText)
		template.AddItem("📏", "总大小", msgelem.FormatSize(info.TotalSize()), msgelem.ItemTypeText)
		
		elapsed := time.Since(p.start)
		template.AddItem("⌚", "总用时", msgelem.FormatDuration(elapsed), msgelem.ItemTypeText)
	}

	text, entities := template.BuildFormattedMessage()

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		peer := &tg.InputPeerUser{UserID: p.ChatID}
		if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, nil); err != nil {
			log.Warn("Failed to edit message for batch task completion", "error", err, "task_id", info.TaskID())
		}
	}
}

func NewProgressTracker(messageID int, chatID int64) ProgressTracker {
	return &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
}
