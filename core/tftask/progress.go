package tftask

import (
	"context"
	"errors"
	"fmt"
	"path"
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
	OnProgress(ctx context.Context, info TaskInfo, downloaded, total int64)
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
	log.FromContext(ctx).Debugf("Progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	
	// 使用新的模板系统
	template := msgelem.NewInfoTemplate("🚀 开始下载", "")
	template.AddItem("📄", "文件名", info.FileName(), msgelem.ItemTypeCode)
	template.AddItem("📂", "保存路径", fmt.Sprintf("[%s]:%s", info.StorageName(), path.Dir(info.StoragePath())), msgelem.ItemTypeCode)
	template.AddItem("📦", "文件大小", msgelem.FormatSize(info.FileSize()), msgelem.ItemTypeText)
	
	text, entities := template.BuildFormattedMessage()
	
	// 添加取消按钮
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
			log.Warn("Failed to edit message for task start", "error", err, "task_id", info.TaskID())
		}
		return
	}
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo, downloaded, total int64) {
	if !shouldUpdateProgress(total, downloaded, int(p.lastUpdatePercent.Load())) {
		return
	}
	percent := int32((downloaded * 100) / total)
	if p.lastUpdatePercent.Load() == percent {
		return
	}
	p.lastUpdatePercent.Store(percent)
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.FileName(), downloaded, total)
	
	// 使用新的模板系统
	template := msgelem.NewProcessingTemplate("正在下载", "")
	
	// 基本信息
	template.AddItem("📄", "文件名", info.FileName(), msgelem.ItemTypeCode)
	template.AddItem("📂", "保存路径", fmt.Sprintf("[%s]:%s", info.StorageName(), path.Dir(info.StoragePath())), msgelem.ItemTypeCode)
	template.AddItem("📦", "文件大小", msgelem.FormatSize(total), msgelem.ItemTypeText)
	
	// 进度信息
	barLength := 12 // 缩短进度条长度，避免消息框过宽
	template.AddProgressBar("📊", "传输进度", downloaded, total, barLength)
	
	// 速度和时间信息
	speed := dlutil.GetSpeed(downloaded, p.start)
	template.AddItem("🚀", "平均速度", msgelem.FormatSize(int64(speed))+"/s", msgelem.ItemTypeText)
	
	elapsed := time.Since(p.start)
	template.AddItem("⌚", "运行时间", msgelem.FormatDuration(elapsed), msgelem.ItemTypeText)
	
	// 计算预计剩余时间
	if speed > 0 {
		remaining := int64(float64(total-downloaded) / speed)
		if remaining > 0 {
			template.AddItem("⏱️", "预计剩余", msgelem.FormatDuration(time.Duration(remaining)*time.Second), msgelem.ItemTypeText)
		}
	}
	
	text, entities := template.BuildFormattedMessage()
	
	// 添加取消和详情按钮
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					tgutil.BuildCancelButton(info.TaskID()),
					tgutil.BuildDetailButton(info.TaskID()),
				},
			},
		},
	}
	
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		peer := &tg.InputPeerUser{UserID: p.ChatID}
		if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, markup); err != nil {
			log.Warn("Failed to edit message for task progress", "error", err, "task_id", info.TaskID())
		}
		return
	}
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	if err != nil {
		log.FromContext(ctx).Errorf("Progress error for file [%s]: %v", info.FileName(), err)
	} else {
		log.FromContext(ctx).Debugf("Progress done for file [%s]", info.FileName())
	}

	var template *msgelem.MessageTemplate
	
	if err != nil {
		if errors.Is(err, context.Canceled) {
			template = msgelem.NewErrorTemplate("任务已取消", "")
			template.AddItem("📄", "文件名", info.FileName(), msgelem.ItemTypeCode)
		} else {
			template = msgelem.NewErrorTemplate("下载失败", "")
			template.AddItem("📄", "文件名", info.FileName(), msgelem.ItemTypeCode)
			template.AddItem("❗", "错误信息", err.Error(), msgelem.ItemTypeText)
		}
	} else {
		template = msgelem.NewSuccessTemplate("下载完成", "")
		template.AddItem("📄", "文件名", info.FileName(), msgelem.ItemTypeCode)
		template.AddItem("📂", "保存路径", fmt.Sprintf("[%s]:%s", info.StorageName(), path.Dir(info.StoragePath())), msgelem.ItemTypeCode)
		
		elapsed := time.Since(p.start)
		template.AddItem("⌚", "总用时", msgelem.FormatDuration(elapsed), msgelem.ItemTypeText)
	}

	text, entities := template.BuildFormattedMessage()

	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		peer := &tg.InputPeerUser{UserID: p.ChatID}
		if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, nil); err != nil {
			log.Warn("Failed to edit message for task completion", "error", err, "task_id", info.TaskID())
		}
	}
}

type ProgressOption func(*Progress)

func NewProgressTrack(
	messageID int,
	chatID int64,
	opts ...ProgressOption,
) ProgressTracker {
	p := &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}
