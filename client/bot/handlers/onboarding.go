package handlers

import (
	"fmt"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/consts"
	"github.com/krau/SaveAny-Bot/storage"
)

// OnboardingStatus 新用户引导状态
type OnboardingStatus struct {
	UserID          int64     `json:"user_id"`
	Step            int       `json:"step"`
	HasDefaultStorage bool    `json:"has_default_storage"`
	HasCustomStorage  bool    `json:"has_custom_storage"`
	HasUsedSave       bool    `json:"has_used_save"`
	CompletedAt       *time.Time `json:"completed_at"`
	LastInteraction   time.Time `json:"last_interaction"`
}

// handleStartCmd 处理智能开始命令
func handleStartCmd(ctx *ext.Context, update *ext.Update) error {
	chatID := update.GetUserChat().GetID()
	
	// 检查用户是否已存在
	user, err := database.GetUserByChatID(ctx, chatID)
	isNewUser := err != nil || user == nil
	
	if isNewUser {
		return handleNewUserOnboarding(ctx, update)
	}
	
	// 检查用户的使用状态，决定显示什么
	return handleReturningUserWelcome(ctx, update, user)
}

// handleNewUserOnboarding 处理新用户引导
func handleNewUserOnboarding(ctx *ext.Context, update *ext.Update) error {
	chatID := update.GetUserChat().GetID()
	
	shortHash := consts.GitCommit
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	
	template := msgelem.NewInfoTemplate("🎉 欢迎使用 SaveAny Bot!", "")
	template.AddItem("🤖", "版本", fmt.Sprintf("%s (%s)", consts.Version, shortHash), msgelem.ItemTypeCode)
	template.AddItem("📁", "功能", "转存 Telegram 文件到各种存储", msgelem.ItemTypeText)
	template.AddItem("⚡", "特色", "支持多种存储类型、智能规则、AI重命名", msgelem.ItemTypeText)
	
	template.AddAction("点击下方按钮开始配置")
	template.SetFooter("💡 完成配置后即可开始使用所有功能")
	
	// 创建引导状态
	onboardingStatus := &OnboardingStatus{
		UserID:          chatID,
		Step:            1,
		LastInteraction: time.Now(),
	}
	
	cacheKey := fmt.Sprintf("onboarding_%d", chatID)
	cache.Set(cacheKey, onboardingStatus)
	
	markup := buildOnboardingStartMarkup()
	
	// 使用格式化消息发送
	text, entities := template.BuildFormattedMessage()
	err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, &ext.ReplyOpts{
		Markup: markup,
	})
	if err != nil {
		// 如果格式化发送失败，回退到普通发送
		ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), &ext.ReplyOpts{
			Markup: markup,
		})
	}
	
	return dispatcher.EndGroups
}

// handleReturningUserWelcome 处理老用户欢迎
func handleReturningUserWelcome(ctx *ext.Context, update *ext.Update, user *database.User) error {
	chatID := user.ChatID
	
	// 分析用户使用情况
	systemStorages := storage.GetUserStorages(ctx, chatID)
	userStorages, _ := database.GetUserStoragesByChatID(ctx, chatID)
	
	hasDefaultStorage := user.DefaultStorage != ""
	totalStorages := len(systemStorages) + len(userStorages)
	
	var template *msgelem.MessageTemplate
	
	if totalStorages == 0 {
		// 没有任何存储配置
		template = msgelem.NewInfoTemplate("👋 欢迎回来！", "看起来你还没有配置任何存储")
		template.AddAction("点击下方按钮开始配置存储")
		markup := buildQuickSetupMarkup()
		
		// 使用格式化消息发送
		text, entities := template.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, &ext.ReplyOpts{
			Markup: markup,
		})
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), &ext.ReplyOpts{
				Markup: markup,
			})
		}
	} else if !hasDefaultStorage {
		// 有存储但没有默认存储
		template = msgelem.NewInfoTemplate("👋 欢迎回来！", "建议设置一个默认存储以便快速保存文件")
		template.AddItem("📁", "可用存储", fmt.Sprintf("共 %d 个", totalStorages), msgelem.ItemTypeText)
		template.AddAction("设置默认存储后可以使用静默模式快速保存")
		markup := buildSetDefaultStorageMarkup()
		
		// 使用格式化消息发送
		text, entities := template.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, &ext.ReplyOpts{
			Markup: markup,
		})
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), &ext.ReplyOpts{
				Markup: markup,
			})
		}
	} else {
		// 配置完整的用户
		template = msgelem.NewInfoTemplate("👋 欢迎回来！", "你的配置看起来很不错")
		template.AddItem("📁", "默认存储", user.DefaultStorage, msgelem.ItemTypeText)
		template.AddItem("⚙️", "存储配置", fmt.Sprintf("共 %d 个", totalStorages), msgelem.ItemTypeText)
		if user.ApplyRule {
			template.AddItem("🎯", "智能规则", "已启用", msgelem.ItemTypeStatus)
		}
		
		template.AddAction("转发文件给我开始保存")
		template.AddAction("使用 /help 查看所有功能")
		
		markup := buildMainFeaturesMarkup()
		
		// 使用格式化消息发送
		text, entities := template.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, &ext.ReplyOpts{
			Markup: markup,
		})
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), &ext.ReplyOpts{
				Markup: markup,
			})
		}
	}
	
	return dispatcher.EndGroups
}

// buildOnboardingStartMarkup 构建引导开始按钮
func buildOnboardingStartMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🚀 开始配置",
						Data: []byte("onboarding_start"),
					},
					&tg.KeyboardButtonCallback{
						Text: "❓ 查看帮助",
						Data: []byte("help_save"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⚡ 跳过引导",
						Data: []byte("onboarding_skip"),
					},
				},
			},
		},
	}
}

// buildQuickSetupMarkup 构建快速设置按钮
func buildQuickSetupMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "➕ 添加存储",
						Data: []byte("storage_add_start"),
					},
					&tg.KeyboardButtonCallback{
						Text: "📋 查看配置",
						Data: []byte("storage_back_to_list"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "❓ 需要帮助",
						Data: []byte("help_storage"),
					},
				},
			},
		},
	}
}

// buildSetDefaultStorageMarkup 构建设置默认存储按钮
func buildSetDefaultStorageMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⭐ 设置默认存储",
						Data: []byte("set_default_storage"),
					},
					&tg.KeyboardButtonCallback{
						Text: "➕ 添加更多存储",
						Data: []byte("storage_add_start"),
					},
				},
			},
		},
	}
}

// buildMainFeaturesMarkup 构建主要功能按钮
func buildMainFeaturesMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📂 文件保存",
						Data: []byte("help_save"),
					},
					&tg.KeyboardButtonCallback{
						Text: "⚙️ 存储管理",
						Data: []byte("storage_back_to_list"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🎯 规则设置",
						Data: []byte("help_rule"),
					},
					&tg.KeyboardButtonCallback{
						Text: "🤖 AI功能",
						Data: []byte("help_ai"),
					},
				},
			},
		},
	}
}

// handleOnboardingCallback 处理引导回调
func handleOnboardingCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	data := string(callback.Data)
	
	switch data {
	case "onboarding_start":
		return handleOnboardingStep1(ctx, update)
	case "onboarding_skip":
		return handleOnboardingSkip(ctx, update)
	case "set_default_storage":
		// 显示存储选择界面用于设置默认存储
		return handleSetDefaultStorageSelection(ctx, update)
	default:
		return dispatcher.EndGroups
	}
}

// handleOnboardingStep1 处理引导第一步：选择存储类型
func handleOnboardingStep1(ctx *ext.Context, update *ext.Update) error {
	template := msgelem.NewInfoTemplate("📋 步骤 1: 添加存储配置", "选择你要使用的存储类型")
	
	template.AddItem("📁", "Alist", "支持多种云盘服务", msgelem.ItemTypeText)
	template.AddItem("🌐", "WebDAV", "标准WebDAV协议", msgelem.ItemTypeText)
	template.AddItem("☁️", "MinIO/S3", "对象存储服务", msgelem.ItemTypeText)
	template.AddItem("💾", "本地存储", "服务器本地磁盘", msgelem.ItemTypeText)
	template.AddItem("📱", "Telegram", "上传到Telegram频道", msgelem.ItemTypeText)
	
	template.AddAction("选择最适合你的存储类型")
	
	markup := msgelem.BuildStorageTypeSelectMarkup()
	
	// 使用格式化消息编辑
	text, entities := template.BuildFormattedMessage()
	callback := update.CallbackQuery
	userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
	err := msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, markup)
	if err != nil {
		// 如果格式化编辑失败，回退到普通编辑
		ctx.EditMessage(callback.Peer.(*tg.PeerUser).UserID, &tg.MessagesEditMessageRequest{
			ID:          callback.MsgID,
			Message:     template.BuildMessage(),
			ReplyMarkup: markup,
		})
	}
	
	return dispatcher.EndGroups
}

// handleOnboardingSkip 处理跳过引导
func handleOnboardingSkip(ctx *ext.Context, update *ext.Update) error {
	chatID := update.GetUserChat().GetID()
	cacheKey := fmt.Sprintf("onboarding_%d", chatID)
	cache.Del(cacheKey)
	
	template := msgelem.NewInfoTemplate("✅ 引导已跳过", "你可以随时使用 /help 查看帮助")
	template.AddAction("转发文件给我开始保存")
	template.AddAction("使用 /storage_list 管理存储配置")
	
	// 使用格式化消息编辑
	text, entities := template.BuildFormattedMessage()
	callback := update.CallbackQuery
	userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
	err := msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, nil)
	if err != nil {
		// 如果格式化编辑失败，回退到普通编辑
		ctx.EditMessage(callback.Peer.(*tg.PeerUser).UserID, &tg.MessagesEditMessageRequest{
			ID:      callback.MsgID,
			Message: template.BuildMessage(),
		})
	}
	
	return dispatcher.EndGroups
}

// checkOnboardingProgress 检查并更新引导进度
func checkOnboardingProgress(ctx *ext.Context, chatID int64, action string) {
	cacheKey := fmt.Sprintf("onboarding_%d", chatID)
	status, exists := cache.Get[*OnboardingStatus](cacheKey)
	if !exists || status.CompletedAt != nil {
		return
	}
	
	// 更新进度
	switch action {
	case "storage_added":
		status.HasCustomStorage = true
	case "default_storage_set":
		status.HasDefaultStorage = true
	case "file_saved":
		status.HasUsedSave = true
	}
	
	status.LastInteraction = time.Now()
	
	// 检查是否完成引导
	if status.HasCustomStorage && status.HasDefaultStorage && status.HasUsedSave {
		now := time.Now()
		status.CompletedAt = &now
		// 可以发送完成引导的祝贺消息
	}
	
	cache.Set(cacheKey, status)
}

// handleSetDefaultStorageSelection 处理设置默认存储的选择界面
func handleSetDefaultStorageSelection(ctx *ext.Context, update *ext.Update) error {
	chatID := update.CallbackQuery.GetUserID()
	
	// 构建选择默认存储的消息
	template := msgelem.NewInfoTemplate("⭐ 设置默认存储", "选择一个存储作为默认保存位置")
	template.AddAction("选择后将用于快速保存和静默模式")
	
	// 获取存储选择的标记
	markup, err := msgelem.BuildSetDefaultStorageMarkup(ctx, chatID)
	if err != nil {
		template = msgelem.NewErrorTemplate("获取存储列表失败", err.Error())
		text, entities := template.BuildFormattedMessage()
		callback := update.CallbackQuery
		userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
		msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, nil)
		return dispatcher.EndGroups
	}
	
	// 使用格式化消息编辑
	text, entities := template.BuildFormattedMessage()
	callback := update.CallbackQuery
	userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
	err = msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, markup)
	if err != nil {
		// 如果格式化编辑失败，回退到普通编辑
		ctx.EditMessage(callback.Peer.(*tg.PeerUser).UserID, &tg.MessagesEditMessageRequest{
			ID:          callback.MsgID,
			Message:     template.BuildMessage(),
			ReplyMarkup: markup,
		})
	}
	
	return dispatcher.EndGroups
}