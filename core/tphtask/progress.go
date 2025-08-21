package tphtask

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
)

type ProgressTracker interface {
	OnStart(ctx context.Context, info TaskInfo)
	OnProgress(ctx context.Context, info TaskInfo)
	OnDone(ctx context.Context, info TaskInfo, err error)
}

type Progress struct {
	MessageID int
	ChatID    int64
}

func (p *Progress) OnStart(ctx context.Context, info TaskInfo) {
	logger := log.FromContext(ctx)
	logger.Debugf("Telegraph task progress tracking started for message %d in chat %d", p.MessageID, p.ChatID)
	
	// 使用新的模板系统
	template := msgelem.NewInfoTemplate("🚀 开始Telegraph下载", "")
	template.AddItem("🖼️", "图片数量", fmt.Sprintf("%d", info.TotalPics()), msgelem.ItemTypeText)
	
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
			log.Warn("Failed to edit message for Telegraph task start", "error", err, "task_id", info.TaskID())
		}
		return
	}
}

func (p *Progress) OnProgress(ctx context.Context, info TaskInfo) {
	if !shouldUpdateProgress(info.Downloaded(), int64(info.TotalPics())) {
		return
	}
	log.FromContext(ctx).Debugf("Progress update: %s, %d/%d", info.TaskID(), info.Downloaded(), info.TotalPics())
	
	// 使用新的模板系统
	template := msgelem.NewProcessingTemplate("Telegraph下载中", "")
	
	// 基本信息
	template.AddItem("🖼️", "图片数量", fmt.Sprintf("%d", info.TotalPics()), msgelem.ItemTypeText)
	
	// 进度信息
	template.AddProgressBar("📊", "下载进度", info.Downloaded(), int64(info.TotalPics()), 12)
	template.AddItem("📏", "已下载", fmt.Sprintf("%d/%d", info.Downloaded(), info.TotalPics()), msgelem.ItemTypeText)
	
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
			log.Warn("Failed to edit message for Telegraph task progress", "error", err, "task_id", info.TaskID())
		}
		return
	}
}

func (p *Progress) OnDone(ctx context.Context, info TaskInfo, err error) {
	logger := log.FromContext(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Infof("Telegraph task %s was canceled", info.TaskID())
			
			template := msgelem.NewErrorTemplate("Telegraph任务已取消", "")
			template.AddItem("🖼️", "图片数量", fmt.Sprintf("%d", info.TotalPics()), msgelem.ItemTypeText)
			
			text, entities := template.BuildFormattedMessage()
			
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				peer := &tg.InputPeerUser{UserID: p.ChatID}
				if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, nil); err != nil {
					log.Warn("Failed to edit message for Telegraph task cancellation", "error", err, "task_id", info.TaskID())
				}
			}
		} else {
			logger.Errorf("Telegraph task %s failed: %s", info.TaskID(), err)
			
			template := msgelem.NewErrorTemplate("Telegraph下载失败", "")
			template.AddItem("🖼️", "图片数量", fmt.Sprintf("%d", info.TotalPics()), msgelem.ItemTypeText)
			template.AddItem("❗", "错误信息", err.Error(), msgelem.ItemTypeText)
			
			text, entities := template.BuildFormattedMessage()
			
			ext := tgutil.ExtFromContext(ctx)
			if ext != nil {
				peer := &tg.InputPeerUser{UserID: p.ChatID}
				if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, nil); err != nil {
					log.Warn("Failed to edit message for Telegraph task failure", "error", err, "task_id", info.TaskID())
				}
			}
		}
		return
	}
	logger.Infof("Telegraph task %s completed successfully", info.TaskID())

	template := msgelem.NewSuccessTemplate("Telegraph下载完成", "")
	template.AddItem("🖼️", "图片数量", fmt.Sprintf("%d", info.TotalPics()), msgelem.ItemTypeText)
	template.AddItem("📂", "保存路径", fmt.Sprintf("[%s]:%s", info.StorageName(), path.Dir(info.StoragePath())), msgelem.ItemTypeCode)
	
	text, entities := template.BuildFormattedMessage()
	
	ext := tgutil.ExtFromContext(ctx)
	if ext != nil {
		peer := &tg.InputPeerUser{UserID: p.ChatID}
		if err := msgelem.EditWithFormattedText(ext, peer, p.MessageID, text, entities, nil); err != nil {
			log.Warn("Failed to edit message for Telegraph task completion", "error", err, "task_id", info.TaskID())
		}
	}
}

func NewProgress(messageID int, chatID int64) *Progress {
	return &Progress{
		MessageID: messageID,
		ChatID:    chatID,
	}
}
