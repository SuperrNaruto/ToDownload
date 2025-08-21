package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/pkg/consts"
)

func handleHelpCmd(ctx *ext.Context, update *ext.Update) error {
	shortHash := consts.GitCommit
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	
	// 构建版本信息
	versionInfo := []msgelem.StatusItem{
		{Name: "版本", Value: consts.Version, Success: true},
		{Name: "提交", Value: shortHash, Success: true},
	}
	
	// 构建格式化消息
	text, entities := msgelem.BuildStatusMessage("Save Any Bot - Telegram文件转存工具", versionInfo)
	
	// 添加提示文本
	additionalText, additionalEntities := msgelem.BuildFormattedMessage(
		styling.Plain("\n💡 选择下方功能分类获取详细帮助："),
	)
	
	// 合并消息
	finalText := text + additionalText
	finalEntities := append(entities, additionalEntities...)
	
	markup := buildHelpMainMarkup()
	
	// 使用新的格式化发送方法
	err := msgelem.ReplyWithFormattedText(ctx, update, finalText, finalEntities, &ext.ReplyOpts{
		Markup: markup,
	})
	if err != nil {
		// 如果格式化发送失败，fallback到普通发送
		fallbackText := fmt.Sprintf(`🤖 Save Any Bot
📁 转存你的 Telegram 文件到各种存储

📊 版本信息
• 版本: %s  
• 提交: %s

💡 选择下方功能分类获取详细帮助：`, consts.Version, shortHash)
		
		ctx.Reply(update, ext.ReplyTextString(fallbackText), &ext.ReplyOpts{
			Markup: markup,
		})
	}
	
	return dispatcher.EndGroups
}

// buildHelpMainMarkup 构建主帮助菜单
func buildHelpMainMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📂 文件保存",
						Data: []byte("help_save"),
					},
					&tg.KeyboardButtonCallback{
						Text: "⚙️ 存储配置",
						Data: []byte("help_storage"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📁 目录管理",
						Data: []byte("help_dir"),
					},
					&tg.KeyboardButtonCallback{
						Text: "🎯 规则设置",
						Data: []byte("help_rule"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🤖 AI功能",
						Data: []byte("help_ai"),
					},
					&tg.KeyboardButtonCallback{
						Text: "👀 监控功能",
						Data: []byte("help_watch"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🔧 高级设置",
						Data: []byte("help_advanced"),
					},
					&tg.KeyboardButtonCallback{
						Text: "❓ 常见问题",
						Data: []byte("help_faq"),
					},
				},
			},
		},
	}
}

// 处理帮助分类回调
func handleHelpCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	data := string(callback.Data)
	
	var helpText string
	var helpEntities []tg.MessageEntityClass
	var markup *tg.ReplyInlineMarkup
	
	switch data {
	case "help_save":
		helpText, helpEntities = buildFormattedSaveHelpText()
		markup = buildHelpBackMarkup()
	case "help_storage":
		helpText, helpEntities = buildFormattedStorageHelpText()
		markup = buildHelpBackMarkup()
	case "help_dir":
		helpText, helpEntities = buildFormattedDirHelpText()
		markup = buildHelpBackMarkup()
	case "help_rule":
		helpText, helpEntities = buildFormattedRuleHelpText()
		markup = buildHelpBackMarkup()
	case "help_ai":
		helpText, helpEntities = buildFormattedAIHelpText()
		markup = buildHelpBackMarkup()
	case "help_watch":
		helpText, helpEntities = buildFormattedWatchHelpText()
		markup = buildHelpBackMarkup()
	case "help_advanced":
		helpText, helpEntities = buildFormattedAdvancedHelpText()
		markup = buildHelpBackMarkup()
	case "help_faq":
		helpText, helpEntities = buildFormattedFAQHelpText()
		markup = buildHelpBackMarkup()
	case "help_back":
		// 返回主菜单 - 复用主命令的逻辑
		shortHash := consts.GitCommit
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		
		versionInfo := []msgelem.StatusItem{
			{Name: "版本", Value: consts.Version, Success: true},
			{Name: "提交", Value: shortHash, Success: true},
		}
		
		text, entities := msgelem.BuildStatusMessage("Save Any Bot - Telegram文件转存工具", versionInfo)
		additionalText, additionalEntities := msgelem.BuildFormattedMessage(
			styling.Plain("\n💡 选择下方功能分类获取详细帮助："),
		)
		
		helpText = text + additionalText
		helpEntities = append(entities, additionalEntities...)
		markup = buildHelpMainMarkup()
	default:
		return dispatcher.EndGroups
	}
	
	// 使用格式化编辑消息
	peer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
	err := msgelem.EditWithFormattedText(ctx, peer, callback.MsgID, helpText, helpEntities, markup)
	
	if err != nil {
		// 如果编辑失败，尝试发送新消息（fallback到纯文本）
		fallbackText := helpText // 使用相同的文本，但没有entities
		msgelem.SendFormattedMessage(ctx, callback.Peer.(*tg.PeerUser).UserID, fallbackText, nil, markup)
	}
	
	return dispatcher.EndGroups
}

// buildHelpBackMarkup 构建返回按钮
func buildHelpBackMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回主菜单",
						Data: []byte("help_back"),
					},
				},
			},
		},
	}
}

func buildFormattedSaveHelpText() (string, []tg.MessageEntityClass) {
	sections := []msgelem.HelpSection{
		{
			Icon:  "🚀",
			Title: "基础使用方法",
			Items: []string{
				"1️⃣ 转发文件到bot",
				"2️⃣ 选择存储位置", 
				"3️⃣ 确认保存",
			},
		},
		{
			Icon:  "💬",
			Title: "命令说明",
			Items: []string{
				"/save - 回复文件消息保存",
				"/save 自定义名称 - 保存并重命名",
				"/save 名称1 名称2 名称3 - 批量保存多个文件",
			},
		},
		{
			Icon:  "🔇",
			Title: "静默模式",
			Items: []string{
				"/silent - 开关静默模式",
				"静默模式下文件直接保存到默认位置",
			},
		},
		{
			Icon:  "📋",
			Title: "支持的文件类型",
			Items: []string{
				"📄 文档、📷 图片、🎵 音频、🎬 视频、📎 所有媒体文件",
			},
		},
	}
	
	return msgelem.BuildHelpMessage("文件保存功能", "快速保存Telegram文件到各种存储", sections)
}

func buildFormattedStorageHelpText() (string, []tg.MessageEntityClass) {
	sections := []msgelem.HelpSection{
		{
			Icon:  "🗃️",
			Title: "存储类型",
			Items: []string{
				"📁 Alist - 支持多种云盘",
				"🌐 WebDAV - 标准WebDAV协议", 
				"☁️ MinIO/S3 - 对象存储服务",
				"💾 本地存储 - 服务器本地磁盘",
				"📱 Telegram - 上传到Telegram频道",
			},
		},
		{
			Icon:  "⚙️",
			Title: "管理命令",
			Items: []string{
				"/storage - 设置默认存储",
				"/storage_list - 管理存储配置",
				"添加、编辑、删除、测试存储",
			},
		},
		{
			Icon:  "📝",
			Title: "配置步骤",
			Items: []string{
				"1️⃣ 选择存储类型",
				"2️⃣ 按提示输入配置信息",
				"3️⃣ 测试连接",
				"4️⃣ 设为默认（可选）",
			},
		},
	}
	
	return msgelem.BuildHelpMessage("存储配置管理", "管理多种存储后端配置", sections)
}

// 为了先测试，创建简化版本的其他帮助函数
func buildFormattedDirHelpText() (string, []tg.MessageEntityClass) {
	return msgelem.BuildFormattedMessage(
		styling.Bold("📁 目录管理功能"),
		styling.Plain("\n\n目录设置：\n• /dir - 管理存储目录\n• 可设置多个常用目录\n• 支持分层目录结构"),
	)
}

func buildFormattedRuleHelpText() (string, []tg.MessageEntityClass) {
	return msgelem.BuildFormattedMessage(
		styling.Bold("🎯 智能规则系统"),
		styling.Plain("\n\n规则功能：\n• 根据文件特征自动选择存储和目录\n• 支持文件名、类型、大小等条件"),
	)
}

func buildFormattedAIHelpText() (string, []tg.MessageEntityClass) {
	return msgelem.BuildFormattedMessage(
		styling.Bold("🤖 AI智能功能"),
		styling.Plain("\n\nAI命令：\n• "),
		styling.Code("/ai_status"),
		styling.Plain(" - 查看AI功能状态\n• "),
		styling.Code("/ai_toggle"),
		styling.Plain(" - 开启/关闭AI重命名"),
	)
}

func buildFormattedWatchHelpText() (string, []tg.MessageEntityClass) {
	return msgelem.BuildFormattedMessage(
		styling.Bold("👀 频道监控功能"),
		styling.Plain("\n\n监控设置：\n• "),
		styling.Code("/watch"),
		styling.Plain(" - 添加监控频道\n• "),
		styling.Code("/unwatch"),
		styling.Plain(" - 取消监控频道"),
	)
}

func buildFormattedAdvancedHelpText() (string, []tg.MessageEntityClass) {
	return msgelem.BuildFormattedMessage(
		styling.Bold("🔧 高级设置选项"),
		styling.Plain("\n\n性能设置：\n• 并发下载数量调整\n• 重试机制配置\n• 流模式开关"),
	)
}

func buildFormattedFAQHelpText() (string, []tg.MessageEntityClass) {
	return msgelem.BuildFormattedMessage(
		styling.Bold("❓ 常见问题解答"),
		styling.Plain("\n\n"),
		styling.Bold("Q: 文件保存失败怎么办？"),
		styling.Plain("\nA: 检查存储配置和网络连接，查看错误提示\n\n"),
		styling.Bold("获取更多帮助："),
		styling.Plain("\n📖 在线文档：https://sabot.unv.app/usage/"),
	)
}
