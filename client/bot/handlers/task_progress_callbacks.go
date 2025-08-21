package handlers

import (
	"strings"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/core"
)

// handleCancelTaskCallback 处理取消任务回调
func handleCancelTaskCallback(ctx *ext.Context, u *ext.Update) error {
	query := u.CallbackQuery

	// 提取任务ID
	data := string(query.Data)
	taskID := strings.TrimPrefix(data, "cancel_task:")

	logger := log.FromContext(ctx)
	logger.Infof("User %d requested to cancel task %s", query.GetUserID(), taskID)

	// 直接尝试取消任务
	err := core.CancelTask(ctx, taskID)
	if err != nil {
		logger.Errorf("Failed to cancel task %s: %v", taskID, err)
		_, err := ctx.AnswerCallback(msgelem.AlertCallbackAnswer(query.GetQueryID(), "❌ 取消任务失败: "+err.Error()))
		return err
	}

	// 发送成功回答
	_, err = ctx.AnswerCallback(msgelem.CallbackAnswer(query.GetQueryID(), "✅ 任务已取消"))
	return err
}

// handleTaskDetailCallback 处理查看任务详情回调
// 由于没有全局状态跟踪，暂时只提供简单反馈
func handleTaskDetailCallback(ctx *ext.Context, u *ext.Update) error {
	query := u.CallbackQuery

	// 提取任务ID
	data := string(query.Data)
	taskID := strings.TrimPrefix(data, "task_detail:")

	logger := log.FromContext(ctx)
	logger.Debugf("User %d requested task detail for %s", query.GetUserID(), taskID)

	// 由于移除了全局跟踪器，详情功能暂时简化
	_, err := ctx.AnswerCallback(msgelem.CallbackAnswer(query.GetQueryID(), "📊 任务详情功能已简化"))
	return err
}
