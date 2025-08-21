package handlers

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
)

// handleAIStatusCmd handles the /ai_status command - shows AI rename service status
func handleAIStatusCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI status command")

	// 构建状态信息结构
	var statusItems []msgelem.StatusItem
	var additionalParts []styling.StyledTextOption
	
	// 检查全局AI配置
	if !config.Cfg.AI.IsEnabled() {
		statusItems = append(statusItems,
			msgelem.StatusItem{Name: "AI重命名功能", Value: "已禁用 (全局配置)", Success: false},
			msgelem.StatusItem{Name: "配置地址", Value: config.Cfg.AI.BaseURL, Success: true},
			msgelem.StatusItem{Name: "模型", Value: config.Cfg.AI.Model, Success: true},
		)
		additionalParts = append(additionalParts,
			styling.Plain("\n⚠️ "),
			styling.Bold("需要在配置文件中启用AI功能"),
		)
	} else {
		statusItems = append(statusItems,
			msgelem.StatusItem{Name: "AI重命名功能", Value: "已启用", Success: true},
			msgelem.StatusItem{Name: "API地址", Value: config.Cfg.AI.BaseURL, Success: true},
			msgelem.StatusItem{Name: "模型", Value: config.Cfg.AI.Model, Success: true},
			msgelem.StatusItem{Name: "超时时间", Value: fmt.Sprintf("%v", config.Cfg.AI.GetTimeout()), Success: true},
			msgelem.StatusItem{Name: "重试次数", Value: fmt.Sprintf("%d", config.Cfg.AI.GetMaxRetries()), Success: true},
		)
		
		// 检查AI服务是否已初始化
		if tgutil.IsRenameServiceInitialized() {
			renameService := tgutil.GetRenameService()
			if renameService != nil && renameService.IsEnabled() {
				statusItems = append(statusItems,
					msgelem.StatusItem{Name: "AI重命名服务", Value: "运行正常", Success: true},
				)
				additionalParts = append(additionalParts,
					styling.Plain("\n"),
					styling.Bold("📝 支持功能:"),
					styling.Plain("\n  • 普通文件智能重命名"),
					styling.Plain("\n  • 相册文件统一重命名"),
					styling.Plain("\n  • 自动回退机制"),
				)
			} else {
				statusItems = append(statusItems,
					msgelem.StatusItem{Name: "AI重命名服务", Value: "未正常运行", Success: false},
				)
			}
		} else {
			statusItems = append(statusItems,
				msgelem.StatusItem{Name: "AI重命名服务", Value: "未初始化", Success: false},
			)
		}
	}
	
	// 构建主要状态消息
	statusText, statusEntities := msgelem.BuildStatusMessage("AI重命名功能状态", statusItems)
	
	// 添加额外信息
	var additionalText string
	var additionalEntities []tg.MessageEntityClass
	if len(additionalParts) > 0 {
		additionalText, additionalEntities = msgelem.BuildFormattedMessage(additionalParts...)
	}
	
	// 添加命令说明
	commandText, commandEntities := msgelem.BuildFormattedMessage(
		styling.Plain("\n\n"),
		styling.Bold("📋 可用命令:"),
		styling.Plain("\n"),
		styling.Code("/ai_status"),
		styling.Plain(" - 查看AI功能状态\n"),
		styling.Code("/ai_toggle"),
		styling.Plain(" - 开启/关闭AI重命名功能"),
	)
	
	// 合并所有部分
	finalText := statusText + additionalText + commandText
	finalEntities := append(append(statusEntities, additionalEntities...), commandEntities...)
	
	// 发送格式化消息
	err := msgelem.ReplyWithFormattedText(ctx, update, finalText, finalEntities, nil)
	if err != nil {
		// Fallback到纯文本
		fallbackMsg := fmt.Sprintf("🤖 AI重命名功能状态\n\n")
		for _, item := range statusItems {
			icon := "✅"
			if !item.Success {
				icon = "❌"
			}
			fallbackMsg += fmt.Sprintf("%s %s: %s\n", icon, item.Name, item.Value)
		}
		ctx.Reply(update, ext.ReplyTextString(fallbackMsg), nil)
	}
	return dispatcher.EndGroups
}


// handleAIToggleCmd handles the /ai_toggle command - toggle AI rename functionality
func handleAIToggleCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI toggle command")

	// 构建当前状态信息
	currentStatus := config.Cfg.AI.IsEnabled()
	var statusItems []msgelem.StatusItem
	
	// 主状态
	statusValue := "已禁用"
	if currentStatus {
		statusValue = "已启用"
	}
	statusItems = append(statusItems,
		msgelem.StatusItem{Name: "AI重命名功能", Value: statusValue, Success: currentStatus},
		msgelem.StatusItem{Name: "API地址", Value: config.Cfg.AI.BaseURL, Success: true},
		msgelem.StatusItem{Name: "模型", Value: config.Cfg.AI.Model, Success: true},
		msgelem.StatusItem{Name: "超时时间", Value: fmt.Sprintf("%v", config.Cfg.AI.GetTimeout()), Success: true},
		msgelem.StatusItem{Name: "重试次数", Value: fmt.Sprintf("%d", config.Cfg.AI.GetMaxRetries()), Success: true},
	)
	
	// 构建状态消息
	statusText, statusEntities := msgelem.BuildStatusMessage("AI功能切换", statusItems)
	
	// 添加操作提示
	promptText, promptEntities := msgelem.BuildFormattedMessage(
		styling.Plain("\n"),
		styling.Bold("请选择操作:"),
	)
	
	// 合并消息
	statusMsg := statusText + promptText
	finalEntities := append(statusEntities, promptEntities...)

	// Create inline keyboard for toggle functionality
	buttons := make([]tg.KeyboardButtonClass, 0)
	
	if currentStatus {
		// If AI is enabled, show disable option
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: "❌ 禁用AI重命名",
			Data: []byte("ai_disable"),
		})
	} else {
		// If AI is disabled, show enable option (only if configuration is valid)
		if config.Cfg.AI.BaseURL != "" && config.Cfg.AI.APIKey != "" && config.Cfg.AI.Model != "" {
			buttons = append(buttons, &tg.KeyboardButtonCallback{
				Text: "✅ 启用AI重命名",
				Data: []byte("ai_enable"),
			})
		} else {
			// 添加警告信息
			warningText, warningEntities := msgelem.BuildFormattedMessage(
				styling.Plain("\n⚠️ "),
				styling.Bold("无法启用：AI配置不完整（缺少API地址、密钥或模型配置）"),
			)
			statusMsg += warningText
			finalEntities = append(finalEntities, warningEntities...)
		}
	}
	
	// Add status check button
	buttons = append(buttons, &tg.KeyboardButtonCallback{
		Text: "🔄 刷新状态",
		Data: []byte("ai_refresh"),
	})
	
	markup := &tg.ReplyInlineMarkup{}
	row := tg.KeyboardButtonRow{}
	row.Buttons = buttons
	markup.Rows = append(markup.Rows, row)

	// 检查是否是回调查询（编辑消息）还是常规命令（新消息）
	if update.CallbackQuery != nil {
		// 编辑现有消息
		peer := &tg.InputPeerUser{UserID: update.CallbackQuery.GetUserID()}
		err := msgelem.EditWithFormattedText(ctx, peer, update.CallbackQuery.GetMsgID(), statusMsg, finalEntities, markup)
		if err != nil {
			// Fallback到简单编辑
			ctx.EditMessage(update.CallbackQuery.GetUserID(), &tg.MessagesEditMessageRequest{
				ID:          update.CallbackQuery.GetMsgID(),
				Message:     statusMsg,
				ReplyMarkup: markup,
			})
		}
	} else {
		// 发送新消息
		err := msgelem.ReplyWithFormattedText(ctx, update, statusMsg, finalEntities, &ext.ReplyOpts{
			Markup: markup,
		})
		if err != nil {
			// Fallback到普通发送
			ctx.Reply(update, ext.ReplyTextString(statusMsg), &ext.ReplyOpts{
				Markup: markup,
			})
		}
	}
	return dispatcher.EndGroups
}


// handleAIToggleCallback handles the callback queries from AI toggle buttons
func handleAIToggleCallback(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI toggle callback")

	callbackData := string(update.CallbackQuery.Data)
	var responseMsg string
	var success bool

	switch callbackData {
	case "ai_enable":
		// Enable AI functionality
		if config.Cfg.AI.BaseURL != "" && config.Cfg.AI.APIKey != "" && config.Cfg.AI.Model != "" {
			config.Cfg.AI.Enable = true
			// Reinitialize AI service with new configuration
			if err := tgutil.InitAIRenameService(ctx, config.Cfg); err != nil {
				logger.Errorf("Failed to initialize AI rename service: %v", err)
				responseMsg = "❌ AI服务初始化失败"
				success = false
			} else {
				responseMsg = "✅ AI重命名功能已启用"
				success = true
				logger.Info("AI rename functionality enabled via bot command")
			}
		} else {
			responseMsg = "❌ 无法启用：AI配置不完整"
			success = false
		}

	case "ai_disable":
		// Disable AI functionality
		config.Cfg.AI.Enable = false
		// Reinitialize AI service as disabled
		if err := tgutil.InitAIRenameService(ctx, config.Cfg); err != nil {
			logger.Errorf("Failed to reinitialize AI rename service as disabled: %v", err)
			responseMsg = "❌ AI服务关闭失败"
			success = false
		} else {
			responseMsg = "❌ AI重命名功能已禁用"
			success = true
			logger.Info("AI rename functionality disabled via bot command")
		}

	case "ai_refresh":
		// Refresh status - rebuild the toggle interface
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID:   update.CallbackQuery.GetQueryID(),
			Message:   "状态已刷新",
			CacheTime: 1,
		})
		return handleAIToggleCmd(ctx, update)

	default:
		responseMsg = "❓ 未知操作"
		success = false
	}

	// Answer the callback query
	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID:   update.CallbackQuery.GetQueryID(),
		Alert:     success,
		Message:   responseMsg,
		CacheTime: 5,
	})

	// If the operation was successful, refresh the toggle interface
	if success && callbackData != "ai_refresh" {
		return handleAIToggleCmd(ctx, update)
	}

	return dispatcher.EndGroups
}

// parseAICommand parses AI command arguments
func parseAICommand(text string) []string {
	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return []string{}
	}
	return parts[1:]
}