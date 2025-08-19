package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	statusMsg += "/ai_test - 测试AI重命名功能"

	ctx.Reply(update, ext.ReplyTextString(statusMsg), nil)
	return dispatcher.EndGroups
}

// handleAITestCmd handles the /ai_test command - tests AI rename functionality
func handleAITestCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI test command")

	// Check if AI is enabled
	if !config.Cfg.AI.IsEnabled() {
		ctx.Reply(update, ext.ReplyTextString("❌ AI重命名功能未启用。请在配置文件中启用AI功能。"), nil)
		return dispatcher.EndGroups
	}

	// Check if AI service is initialized and working
	if !tgutil.IsRenameServiceInitialized() {
		ctx.Reply(update, ext.ReplyTextString("❌ AI重命名服务未初始化。请重启应用程序。"), nil)
		return dispatcher.EndGroups
	}

	renameService := tgutil.GetRenameService()
	if renameService == nil || !renameService.IsEnabled() {
		ctx.Reply(update, ext.ReplyTextString("❌ AI重命名服务不可用。请检查配置。"), nil)
		return dispatcher.EndGroups
	}

	// Send testing message
	testMsg, err := ctx.Reply(update, ext.ReplyTextString("🔄 正在测试AI重命名功能..."), nil)
	if err != nil {
		logger.Errorf("Failed to send test message: %s", err)
		return dispatcher.EndGroups
	}

	// Test normal file renaming
	testCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	testFilename := "IMG_20240315_142530.jpg"
	testMessage := "这是一张在北京故宫拍摄的春日樱花照片，记录了美好的旅行时光。"

	logger.Info("Testing AI rename", "test_file", testFilename, "test_message", testMessage)

	result, err := renameService.RenameFile(testCtx, testFilename, testMessage)
	if err != nil {
		errorMsg := fmt.Sprintf("❌ AI重命名测试失败:\n\n%s\n\n🔧 请检查:\n• API密钥是否正确\n• 网络连接是否正常\n• API地址是否可访问", err.Error())
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:      testMsg.ID,
			Message: errorMsg,
		})
		return dispatcher.EndGroups
	}

	// Test album renaming
	albumResult, err := renameService.RenameAlbum(testCtx, "今天和朋友们一起去颐和园游玩，拍了很多漂亮的风景照片")
	if err != nil {
		logger.Warn("Album rename test failed", "error", err)
		albumResult = "相册测试失败"
	}

	// Display results
	resultMsg := "✅ AI重命名测试成功!\n\n"
	resultMsg += "📝 普通文件测试:\n"
	resultMsg += fmt.Sprintf("原文件名: %s\n", testFilename)
	resultMsg += fmt.Sprintf("消息内容: %s\n", testMessage)
	resultMsg += fmt.Sprintf("重命名结果: %s\n\n", result)
	resultMsg += "📁 相册文件测试:\n"
	resultMsg += "消息内容: 今天和朋友们一起去颐和园游玩，拍了很多漂亮的风景照片\n"
	resultMsg += fmt.Sprintf("基础名称: %s\n\n", albumResult)
	resultMsg += "🎯 重命名格式说明:\n"
	resultMsg += "• 普通文件: 名称.作者.时间.要点\n"
	resultMsg += "• 相册文件: 统一名称_序号\n"
	resultMsg += "• 各部分根据内容自动提取"

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:      testMsg.ID,
		Message: resultMsg,
	})

	return dispatcher.EndGroups
}

// handleAIHelpCmd handles the /ai_help command - shows AI feature help
func handleAIHelpCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	logger.Debug("Processing AI help command")

	helpMsg := "🤖 AI智能重命名功能帮助\n\n"
	helpMsg += "📝 功能介绍:\n"
	helpMsg += "AI重命名功能可以根据文件内容和消息信息，自动生成语义化的文件名，让您的文件管理更加有序。\n\n"
	
	helpMsg += "🎯 重命名规则:\n"
	helpMsg += "• 普通文件格式: 名称.作者.时间.要点\n"
	helpMsg += "  示例: 北京故宫.张三.2024年3月.春日樱花\n"
	helpMsg += "• 相册文件格式: 统一名称_序号\n"
	helpMsg += "  示例: 颐和园游玩_01, 颐和园游玩_02\n\n"
	
	helpMsg += "🔧 工作方式:\n"
	helpMsg += "1. 分析原始文件名和消息内容\n"
	helpMsg += "2. 使用AI提取关键信息（名称、作者、时间、要点）\n"
	helpMsg += "3. 按格式生成新文件名\n"
	helpMsg += "4. 如果AI失败，自动使用备用命名方式\n\n"
	
	helpMsg += "⚡ 自动触发:\n"
	helpMsg += "• 发送媒体文件时自动重命名\n"
	helpMsg += "• 发送相册时统一重命名\n"
	helpMsg += "• 使用 /save 命令保存时重命名\n\n"
	
	helpMsg += "📋 可用命令:\n"
	helpMsg += "/ai_status - 查看AI功能状态\n"
	helpMsg += "/ai_test - 测试AI重命名功能\n"
	helpMsg += "/ai_help - 显示此帮助信息\n\n"
	
	helpMsg += "💡 提示:\n"
	helpMsg += "• 消息内容越详细，重命名效果越好\n"
	helpMsg += "• 支持中文和英文内容识别\n"
	helpMsg += "• 自动过滤无效字符，确保文件名合规"

	ctx.Reply(update, ext.ReplyTextString(helpMsg), nil)
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