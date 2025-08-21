package handlers

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/core"
)

func handleCancelCallback(ctx *ext.Context, update *ext.Update) error {
	parts := strings.Split(string(update.CallbackQuery.Data), " ")
	if len(parts) < 2 {
		// 如果是简单的 "cancel" 回调，清理相关缓存并关闭消息
		chatID := update.GetUserChat().GetID()

		// 清理存储配置向导缓存
		wizardKey := fmt.Sprintf("storage_wizard_%d", chatID)
		cache.Del(wizardKey)

		// 清理存储名称输入缓存
		nameInputKey := fmt.Sprintf("storage_name_input_%d", chatID)
		cache.Del(nameInputKey)

		// 编辑消息
		ctx.EditMessage(chatID, &tg.MessagesEditMessageRequest{
			ID:      update.CallbackQuery.GetMsgID(),
			Message: "✅ 操作已取消",
		})

		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), "操作已取消"))
		return dispatcher.EndGroups
	}
	taskid := parts[1]
	if err := core.CancelTask(ctx, taskid); err != nil {
		log.FromContext(ctx).Errorf("error cancelling task %s: %v", taskid, err)
		ctx.AnswerCallback(msgelem.AlertCallbackAnswer(update.CallbackQuery.GetQueryID(), "取消任务失败: "+err.Error()))
		return dispatcher.EndGroups
	}

	ctx.EditMessage(update.CallbackQuery.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: "正在取消任务...",
	})

	return dispatcher.EndGroups
}
