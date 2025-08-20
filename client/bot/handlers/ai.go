package handlers

import (
	"fmt"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
)

// handleAIStatusCmd handles the /ai_status command - shows AI rename service status
func handleAIStatusCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI status command")

	var statusMsg string
	
	// Check global AI configuration
	if !config.Cfg.AI.IsEnabled() {
		statusMsg = "🤖 AI重命名功能状态: 已禁用 (全局配置)\n\n"
		statusMsg += fmt.Sprintf("📍 配置地址: %s\n", config.Cfg.AI.BaseURL)
		statusMsg += fmt.Sprintf("🤖 模型: %s\n", config.Cfg.AI.Model)
		statusMsg += "⚠️ 需要在配置文件中启用AI功能"
	} else {
		statusMsg = "🤖 AI重命名功能状态: 已启用 ✅\n\n"
		statusMsg += fmt.Sprintf("📍 API地址: %s\n", config.Cfg.AI.BaseURL)
		statusMsg += fmt.Sprintf("🤖 模型: %s\n", config.Cfg.AI.Model)
		statusMsg += fmt.Sprintf("⏰ 超时时间: %v\n", config.Cfg.AI.GetTimeout())
		statusMsg += fmt.Sprintf("🔄 重试次数: %d\n", config.Cfg.AI.GetMaxRetries())
		
		// Check if AI service is initialized
		if tgutil.IsRenameServiceInitialized() {
			renameService := tgutil.GetRenameService()
			if renameService != nil && renameService.IsEnabled() {
				statusMsg += "\n✅ AI重命名服务: 运行正常\n"
				statusMsg += "📝 支持功能:\n"
				statusMsg += "  • 普通文件智能重命名\n"
				statusMsg += "  • 相册文件统一重命名\n"
				statusMsg += "  • 自动回退机制"
			} else {
				statusMsg += "\n⚠️ AI重命名服务: 未正常运行"
			}
		} else {
			statusMsg += "\n⚠️ AI重命名服务: 未初始化"
		}
	}
	
	statusMsg += "\n\n📋 可用命令:\n"
	statusMsg += "/ai_status - 查看AI功能状态\n"
	statusMsg += "/ai_toggle - 开启/关闭AI重命名功能"

	ctx.Reply(update, ext.ReplyTextString(statusMsg), nil)
	return dispatcher.EndGroups
}


// handleAIToggleCmd handles the /ai_toggle command - toggle AI rename functionality
func handleAIToggleCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI toggle command")

	// Build current status message
	var statusMsg string
	currentStatus := config.Cfg.AI.IsEnabled()
	
	if currentStatus {
		statusMsg = "🤖 AI重命名功能: 当前已启用 ✅\n\n"
	} else {
		statusMsg = "🤖 AI重命名功能: 当前已禁用 ❌\n\n"
	}
	
	statusMsg += fmt.Sprintf("📍 API地址: %s\n", config.Cfg.AI.BaseURL)
	statusMsg += fmt.Sprintf("🤖 模型: %s\n", config.Cfg.AI.Model)
	statusMsg += fmt.Sprintf("⏰ 超时时间: %v\n", config.Cfg.AI.GetTimeout())
	statusMsg += fmt.Sprintf("🔄 重试次数: %d\n\n", config.Cfg.AI.GetMaxRetries())
	
	statusMsg += "请选择操作:"

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
			statusMsg += "\n⚠️ 无法启用：AI配置不完整（缺少API地址、密钥或模型配置）"
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

	// Check if this is a callback query (edit message) or regular command (new message)
	if update.CallbackQuery != nil {
		// Edit existing message
		ctx.EditMessage(update.CallbackQuery.GetUserID(), &tg.MessagesEditMessageRequest{
			ID:          update.CallbackQuery.GetMsgID(),
			Message:     statusMsg,
			ReplyMarkup: markup,
		})
	} else {
		// Send new message
		ctx.Reply(update, ext.ReplyTextString(statusMsg), &ext.ReplyOpts{
			Markup: markup,
		})
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