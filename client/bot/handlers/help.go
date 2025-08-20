package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/pkg/consts"
)

func handleHelpCmd(ctx *ext.Context, update *ext.Update) error {
	shortHash := consts.GitCommit
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	
	helpText := fmt.Sprintf(`🤖 **Save Any Bot**
📁 转存你的 Telegram 文件到各种存储

📊 **版本信息**
• 版本: %s
• 提交: %s

💡 选择下方功能分类获取详细帮助：`, consts.Version, shortHash)

	markup := buildHelpMainMarkup()
	ctx.Reply(update, ext.ReplyTextString(helpText), &ext.ReplyOpts{
		Markup: markup,
	})
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
	var backButton bool = true
	
	switch data {
	case "help_save":
		helpText = buildSaveHelpText()
	case "help_storage":
		helpText = buildStorageHelpText()
	case "help_dir":
		helpText = buildDirHelpText()
	case "help_rule":
		helpText = buildRuleHelpText()
	case "help_ai":
		helpText = buildAIHelpText()
	case "help_watch":
		helpText = buildWatchHelpText()
	case "help_advanced":
		helpText = buildAdvancedHelpText()
	case "help_faq":
		helpText = buildFAQHelpText()
	case "help_back":
		return handleHelpCmd(ctx, update)
	default:
		return dispatcher.EndGroups
	}
	
	markup := buildHelpBackMarkup()
	if !backButton {
		markup = nil
	}
	
	ctx.EditMessage(callback.Peer.(*tg.PeerUser).UserID, &tg.MessagesEditMessageRequest{
		ID:          callback.MsgID,
		Message:     helpText,
		ReplyMarkup: markup,
	})
	
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

func buildSaveHelpText() string {
	return `📂 **文件保存功能**

**基础使用方法：**
1️⃣ 转发文件到bot
2️⃣ 选择存储位置
3️⃣ 确认保存

**命令说明：**
• /save - 回复文件消息保存
• /save 自定义名称 - 保存并重命名
• /save 名称1 名称2 名称3 - 批量保存多个文件

**静默模式：**
• /silent - 开关静默模式
• 静默模式下文件直接保存到默认位置

**支持的文件类型：**
📄 文档、📷 图片、🎵 音频、🎬 视频、📎 所有媒体文件`
}

func buildStorageHelpText() string {
	return `⚙️ **存储配置管理**

**存储类型：**
• 📁 **Alist** - 支持多种云盘
• 🌐 **WebDAV** - 标准WebDAV协议
• ☁️ **MinIO/S3** - 对象存储服务
• 💾 **本地存储** - 服务器本地磁盘
• 📱 **Telegram** - 上传到Telegram频道

**管理命令：**
• /storage - 设置默认存储
• /storage_list - 管理存储配置
• 添加、编辑、删除、测试存储

**配置步骤：**
1️⃣ 选择存储类型
2️⃣ 按提示输入配置信息
3️⃣ 测试连接
4️⃣ 设为默认（可选）`
}

func buildDirHelpText() string {
	return `📁 **目录管理功能**

**目录设置：**
• /dir - 管理存储目录
• 可设置多个常用目录
• 支持分层目录结构

**使用方式：**
• 保存文件时选择目录
• 规则自动分配目录
• 默认根目录保存

**目录操作：**
• ➕ 添加新目录
• ✏️ 编辑目录路径
• 🗑️ 删除目录
• 📌 设为默认目录`
}

func buildRuleHelpText() string {
	return `🎯 **智能规则系统**

**规则功能：**
• 根据文件特征自动选择存储和目录
• 支持文件名、类型、大小等条件
• 可设置优先级和多重条件

**规则管理：**
• /rule - 管理规则设置
• 添加、编辑、删除规则
• 启用/禁用规则

**规则类型：**
• 📄 文件扩展名匹配
• 📏 文件大小范围
• 🏷️ 文件名关键词
• 📁 发送者/频道匹配`
}

func buildAIHelpText() string {
	return `🤖 **AI智能功能**

**文件重命名：**
• 使用AI分析文件内容智能重命名
• 支持图片、视频、文档等类型
• 保持文件扩展名不变

**AI命令：**
• /ai_status - 查看AI功能状态
• /ai_toggle - 开启/关闭AI重命名

**命名规则：**
• 普通文件：名称.作者.时间.要点
• 相册文件：统一名称_序号
• 失败时使用原文件名

**注意事项：**
• 需要配置AI服务API
• 处理时间较长请耐心等待`
}

func buildWatchHelpText() string {
	return `👀 **频道监控功能**

**监控设置：**
• /watch - 添加监控频道
• /unwatch - 取消监控频道
• 自动保存频道新文件

**监控条件：**
• 支持正则表达式过滤
• 可设置文件类型过滤
• 按规则自动分类保存

**使用场景：**
• 备份重要频道内容
• 收集特定类型文件
• 自动整理频道资源`
}

func buildAdvancedHelpText() string {
	return `🔧 **高级设置选项**

**性能设置：**
• 并发下载数量调整
• 重试机制配置
• 流模式开关

**安全设置：**
• 用户权限管理
• 访问控制设置
• 日志记录级别

**系统信息：**
• 存储使用状况
• 任务队列状态
• 系统运行状态

**配置文件：**
• 修改config.toml进行高级配置
• 重启服务生效`
}

func buildFAQHelpText() string {
	return `❓ **常见问题解答**

**Q: 文件保存失败怎么办？**
A: 检查存储配置和网络连接，查看错误提示

**Q: 如何批量保存文件？**
A: 使用 /save 名称1 名称2 或开启静默模式

**Q: AI重命名不工作？**
A: 检查AI服务配置和API密钥设置

**Q: 存储空间不足？**
A: 清理无用文件或添加新的存储配置

**Q: 如何备份配置？**
A: 导出config.toml文件和数据库文件

**获取更多帮助：**
📖 在线文档：https://sabot.unv.app/usage/`
}
