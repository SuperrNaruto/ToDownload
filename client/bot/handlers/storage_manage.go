package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/configval"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/storage"
)

// handleStorageListCmd 处理 /storage_list 命令
func handleStorageListCmd(ctx *ext.Context, update *ext.Update) error {
	chatID := update.GetUserChat().GetID()

	template := msgelem.NewInfoTemplate("存储配置管理", "查看和管理你的所有存储配置")

	// 获取系统配置的存储
	systemStorages := storage.GetUserStorages(ctx, chatID)
	if len(systemStorages) > 0 {
		template.AddItem("🏢", "系统存储", fmt.Sprintf("共 %d 个", len(systemStorages)), msgelem.ItemTypeText)
		for i, stor := range systemStorages {
			if i < 3 { // 只显示前3个，避免消息过长
				template.AddItem("  🟢", stor.Name(), string(stor.Type()), msgelem.ItemTypeText)
			}
		}
		if len(systemStorages) > 3 {
			template.AddItem("  📋", "更多", fmt.Sprintf("还有 %d 个系统存储", len(systemStorages)-3), msgelem.ItemTypeText)
		}
	}

	// 获取用户自定义存储配置
	userStorages, err := database.GetUserStoragesByChatID(ctx, chatID)
	if err != nil {
		errorTemplate := msgelem.NewErrorTemplate("获取存储列表失败", err.Error())

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return nil
	}

	if len(userStorages) > 0 {
		template.AddItem("👤", "自定义存储", fmt.Sprintf("共 %d 个", len(userStorages)), msgelem.ItemTypeText)
		for _, userStorage := range userStorages {
			statusIcon := "🟢"
			statusText := "启用"
			if !userStorage.Enable {
				statusIcon = "🔴"
				statusText = "禁用"
			}

			template.AddItem("  "+statusIcon, userStorage.Name, fmt.Sprintf("%s (%s)", userStorage.Type, statusText), msgelem.ItemTypeText)
		}
	} else {
		if len(systemStorages) == 0 {
			template.AddAction("暂无可用存储，请添加存储配置")
		} else {
			template.AddAction("点击下方按钮添加自定义存储配置")
		}
	}

	// 总是显示操作按钮
	markup, err := msgelem.BuildStorageManageMarkup(ctx, userStorages)
	if err != nil {
		// 如果获取标记失败，使用格式化发送
		text, entities := template.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), nil)
		}
		return nil
	}

	// 使用格式化消息发送
	text, entities := template.BuildFormattedMessage()
	err = msgelem.ReplyWithFormattedText(ctx, update, text, entities, &ext.ReplyOpts{
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

// startStorageConfigWizard 开始存储配置向导
func startStorageConfigWizard(ctx *ext.Context, update *ext.Update, storageName, storageType, description string) error {
	var template *msgelem.MessageTemplate
	var expectedFields []string

	switch storageType {
	case "alist":
		template = msgelem.NewInfoTemplate("配置 Alist 存储", "请按照下面的格式发送配置信息")
		template.AddItem("📝", "格式", "URL,用户名,密码[,base_path]", msgelem.ItemTypeCode)
		template.AddItem("💡", "示例", "https://alist.example.com,admin,password123,/upload", msgelem.ItemTypeCode)
		template.AddItem("🌐", "URL", "Alist 服务器地址", msgelem.ItemTypeText)
		template.AddItem("👤", "用户名", "登录用户名", msgelem.ItemTypeText)
		template.AddItem("🔐", "密码", "登录密码", msgelem.ItemTypeText)
		template.AddItem("📁", "路径", "基础存储路径 (可选，默认为 /)", msgelem.ItemTypeText)
		expectedFields = []string{"url", "username", "password", "base_path"}

	case "webdav":
		template = msgelem.NewInfoTemplate("配置 WebDAV 存储", "请按照下面的格式发送配置信息")
		template.AddItem("📝", "格式", "URL,用户名,密码[,路径]", msgelem.ItemTypeCode)
		template.AddItem("💡", "示例", "https://webdav.example.com,user,pass123,/files", msgelem.ItemTypeCode)
		template.AddItem("🌐", "URL", "WebDAV 服务器地址", msgelem.ItemTypeText)
		template.AddItem("👤", "用户名", "登录用户名", msgelem.ItemTypeText)
		template.AddItem("🔐", "密码", "登录密码", msgelem.ItemTypeText)
		template.AddItem("📁", "路径", "存储路径 (可选，默认为 /)", msgelem.ItemTypeText)
		expectedFields = []string{"url", "username", "password", "path"}

	case "minio":
		template = msgelem.NewInfoTemplate("配置 MinIO/S3 存储", "请按照下面的格式发送配置信息")
		template.AddItem("📝", "格式", "endpoint,access_key,secret_key,bucket[,region]", msgelem.ItemTypeCode)
		template.AddItem("💡", "示例", "s3.amazonaws.com,KEY123,SECRET456,my-bucket,us-east-1", msgelem.ItemTypeCode)
		template.AddItem("🌐", "端点", "S3端点地址", msgelem.ItemTypeText)
		template.AddItem("🔑", "访问密钥", "访问密钥ID", msgelem.ItemTypeText)
		template.AddItem("🔐", "秘密密钥", "秘密访问密钥", msgelem.ItemTypeText)
		template.AddItem("🪣", "存储桶", "存储桶名称", msgelem.ItemTypeText)
		template.AddItem("🌍", "区域", "区域 (可选，默认为 us-east-1)", msgelem.ItemTypeText)
		expectedFields = []string{"endpoint", "access_key", "secret_key", "bucket", "region"}

	case "local":
		template = msgelem.NewInfoTemplate("配置本地存储", "请发送本地存储目录路径")
		template.AddItem("📝", "格式", "路径", msgelem.ItemTypeCode)
		template.AddItem("💡", "示例", "/home/user/downloads", msgelem.ItemTypeCode)
		template.AddItem("📁", "路径", "本地存储目录的绝对路径", msgelem.ItemTypeText)
		expectedFields = []string{"base_path"}

	case "telegram":
		template = msgelem.NewInfoTemplate("配置 Telegram 存储", "请发送目标频道或群组的ID")
		template.AddItem("📝", "格式", "chat_id", msgelem.ItemTypeCode)
		template.AddItem("💡", "示例", "-1001234567890", msgelem.ItemTypeCode)
		template.AddItem("📱", "频道ID", "目标频道或群组的ID (负数)", msgelem.ItemTypeText)
		template.AddAction("获取频道ID: 转发频道消息给 @userinfobot")
		expectedFields = []string{"chat_id"}

	default:
		errorTemplate := msgelem.NewErrorTemplate("不支持的存储类型", "请选择支持的存储类型")

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}

	// 存储配置向导状态
	wizardData := tcbdata.StorageConfigWizard{
		ChatID:         update.GetUserChat().GetID(),
		StorageName:    storageName,
		StorageType:    storageType,
		Description:    description,
		ExpectedFields: expectedFields,
	}

	// 使用固定的缓存键，每个用户同时只能配置一个存储
	dataID := fmt.Sprintf("storage_wizard_%d", wizardData.ChatID)
	if err := cache.Set(dataID, wizardData); err != nil {
		errorTemplate := msgelem.NewErrorTemplate("缓存设置失败", "请重试配置过程")
		// 检查是否是回调查询，如果是则编辑消息，否则回复
		if update.CallbackQuery != nil {
			// 使用格式化消息编辑
			text, entities := errorTemplate.BuildFormattedMessage()
			callback := update.CallbackQuery
			userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
			err := msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, nil)
			if err != nil {
				// 如果格式化编辑失败，回退到普通编辑
				ctx.EditMessage(update.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
					ID:      update.CallbackQuery.GetMsgID(),
					Message: errorTemplate.BuildMessage(),
				})
			}
		} else {
			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if err != nil {
				// 如果格式化发送失败，回退到普通发送
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
		}
		return dispatcher.EndGroups
	}

	template.AddAction("发送 /cancel 取消配置")

	// 检查是否是回调查询，如果是则编辑消息，否则回复
	if update.CallbackQuery != nil {
		// 使用格式化消息编辑
		text, entities := template.BuildFormattedMessage()
		callback := update.CallbackQuery
		userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
		err := msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, nil)
		if err != nil {
			// 如果格式化编辑失败，回退到普通编辑
			ctx.EditMessage(update.GetUserChat().GetID(), &tg.MessagesEditMessageRequest{
				ID:      update.CallbackQuery.GetMsgID(),
				Message: template.BuildMessage(),
			})
		}
	} else {
		// 使用格式化消息发送
		text, entities := template.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), nil)
		}
	}
	return dispatcher.EndGroups
}

// handleStorageConfigResponse 处理存储配置响应
func handleStorageConfigResponse(ctx *ext.Context, update *ext.Update) error {
	if update.EffectiveMessage == nil {
		return nil // 继续传递给其他处理器
	}

	text := update.EffectiveMessage.GetMessage()
	if text == "" {
		return nil // 继续传递给其他处理器
	}

	chatID := update.GetUserChat().GetID()

	// 检查是否是取消命令
	if text == "/cancel" {
		// 清理所有相关的缓存
		clearStorageWizardCache(chatID)
		// 清理存储名称输入缓存
		nameInputKey := fmt.Sprintf("storage_name_input_%d", chatID)
		cache.Del(nameInputKey)
		successTemplate := msgelem.NewSuccessTemplate("存储配置已取消", "配置过程已被用户取消")

		// 使用格式化消息发送
		text, entities := successTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			// 如果格式化发送失败，回退到普通发送
			ctx.Reply(update, ext.ReplyTextString(successTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}

	// 处理存储名称输入
	nameInputKey := fmt.Sprintf("storage_name_input_%d", chatID)
	if storageType, ok := cache.Get[string](nameInputKey); ok {
		log.Printf("处理存储名称输入: 用户=%d, 存储类型=%s, 输入内容=%s", chatID, storageType, text)

		// 用户正在输入存储名称
		storageName := strings.TrimSpace(text)

		// 验证存储名称
		if storageName == "" {
			log.Printf("存储名称为空: 用户=%d", chatID)
			errorTemplate := msgelem.NewErrorTemplate("输入无效", "存储名称不能为空，请重新输入")

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if err != nil {
				// 如果格式化发送失败，回退到普通发送
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}

		// 检查存储名称是否已存在
		user, err := database.GetUserByChatID(ctx, chatID)
		if err != nil {
			log.Printf("获取用户信息失败: 用户=%d, 错误=%v", chatID, err)
			errorTemplate := msgelem.NewErrorTemplate("获取用户信息失败", err.Error())

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if err != nil {
				// 如果格式化发送失败，回退到普通发送
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}

		existingStorage, err := database.GetUserStorageByUserIDAndName(ctx, user.ID, storageName)
		if err != nil && err.Error() != "record not found" {
			log.Printf("检查存储名称失败: 用户=%d, 存储名称=%s, 错误=%v", chatID, storageName, err)
			errorTemplate := msgelem.NewErrorTemplate("检查存储名称失败", err.Error())

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if err != nil {
				// 如果格式化发送失败，回退到普通发送
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}

		if existingStorage != nil {
			log.Printf("存储名称已存在: 用户=%d, 存储名称=%s", chatID, storageName)
			errorTemplate := msgelem.NewErrorTemplate("存储名称冲突", fmt.Sprintf("存储名称 '%s' 已存在，请选择其他名称", storageName))

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if err != nil {
				// 如果格式化发送失败，回退到普通发送
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}

		// 清理名称输入缓存
		cache.Del(nameInputKey)
		log.Printf("开始配置向导: 用户=%d, 存储名称=%s, 存储类型=%s", chatID, storageName, storageType)

		// 开始配置向导
		return startStorageConfigWizard(ctx, update, storageName, storageType, "")
	}

	// 查找活跃的配置向导
	wizardData, dataID := findActiveStorageWizard(chatID)
	if wizardData == nil {
		// 没有找到活跃的配置向导，可能是其他消息
		log.Printf("未找到活跃的存储配置向导: 用户=%d, 消息内容=%s", chatID, text)
		return nil // 继续传递给其他处理器
	}

	// 添加调试信息
	log.Printf("找到活跃向导: 用户=%d, 存储名称=%s, 类型=%s", chatID, wizardData.StorageName, wizardData.StorageType)

	// 使用新的验证系统解析配置数据
	log.Printf("开始解析配置: 类型=%s, 内容=%s", wizardData.StorageType, text)

	validator := configval.NewConfigValidator()
	var validationResult *configval.ValidationResult
	var configData map[string]string

	switch wizardData.StorageType {
	case "alist":
		validationResult, configData = validator.ValidateAlistConfig(text)
	case "webdav":
		validationResult, configData = validator.ValidateWebDAVConfig(text)
	case "minio":
		validationResult, configData = validator.ValidateMinIOConfig(text)
	case "local":
		validationResult, configData = validator.ValidateLocalConfig(text)
	case "telegram":
		validationResult, configData = validator.ValidateTelegramConfig(text)
	default:
		errorTemplate := msgelem.NewErrorTemplate("不支持的存储类型", wizardData.StorageType)

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}

	if !validationResult.IsValid {
		log.Printf("配置验证失败: %s", validationResult.Error)
		errorTemplate := msgelem.NewErrorTemplate("配置验证失败", validationResult.Error)
		if validationResult.Suggestion != "" {
			errorTemplate.AddAction(validationResult.Suggestion)
		}

		// 提供智能建议
		suggestions := validator.GetSmartSuggestions(wizardData.StorageType, text)
		for _, suggestion := range suggestions {
			errorTemplate.AddAction(suggestion)
		}

		errorTemplate.AddAction("发送 /cancel 取消配置")

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}

	log.Printf("配置验证成功: %+v", configData)

	// 转换为JSON字符串
	configJSON, err := json.Marshal(configData)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("❌ 配置数据处理失败"), nil)
		return dispatcher.EndGroups
	}

	// 验证配置有效性
	if err := database.ValidateStorageConfig(wizardData.StorageType, string(configJSON)); err != nil {
		errorTemplate := msgelem.NewErrorTemplate("配置验证失败", err.Error())

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}

	// 获取用户信息
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("❌ 获取用户信息失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	// 检查是否是编辑现有配置
	existingStorage, err := database.GetUserStorageByUserIDAndName(ctx, user.ID, wizardData.StorageName)
	if err != nil && err.Error() != "record not found" {
		log.Printf("检查存储配置失败: %v", err)
		ctx.Reply(update, ext.ReplyTextString("❌ 检查存储配置失败: "+err.Error()), nil)
		return dispatcher.EndGroups
	}

	if existingStorage != nil {
		log.Printf("更新现有存储: %s", wizardData.StorageName)
		// 更新现有配置
		existingStorage.Config = string(configJSON)
		existingStorage.Description = wizardData.Description
		existingStorage.Type = wizardData.StorageType

		if err := database.UpdateUserStorage(ctx, existingStorage); err != nil {
			ctx.Reply(update, ext.ReplyTextString("❌ 更新存储配置失败: "+err.Error()), nil)
			return dispatcher.EndGroups
		}

		successTemplate := msgelem.NewSuccessTemplate("存储配置更新成功", fmt.Sprintf("存储 '%s' 配置已更新", wizardData.StorageName))
		configPreview := validator.FormatConfigPreview(configData, true)
		successTemplate.AddItem("📁", "存储名称", wizardData.StorageName, msgelem.ItemTypeText)
		successTemplate.AddItem("🔧", "存储类型", wizardData.StorageType, msgelem.ItemTypeText)
		for key, value := range configPreview {
			successTemplate.AddItem("⚙️", key, value, msgelem.ItemTypeText)
		}
		successTemplate.AddAction("使用 /storage_list 查看所有存储配置")
		ctx.Reply(update, ext.ReplyTextString(successTemplate.BuildMessage()), nil)
	} else {
		log.Printf("创建新存储: %s", wizardData.StorageName)
		// 创建新配置
		userStorage := &database.UserStorage{
			UserID:      user.ID,
			Name:        wizardData.StorageName,
			Type:        wizardData.StorageType,
			Enable:      true, // 默认启用
			Config:      string(configJSON),
			Description: wizardData.Description,
		}

		if err := database.CreateUserStorage(ctx, userStorage); err != nil {
			errorTemplate := msgelem.NewErrorTemplate("创建存储配置失败", err.Error())

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if err != nil {
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}

		successTemplate := msgelem.NewSuccessTemplate("存储配置创建成功", fmt.Sprintf("存储 '%s' 已成功创建并启用", wizardData.StorageName))
		configPreview := validator.FormatConfigPreview(configData, true)
		successTemplate.AddItem("📁", "存储名称", wizardData.StorageName, msgelem.ItemTypeText)
		successTemplate.AddItem("🔧", "存储类型", wizardData.StorageType, msgelem.ItemTypeText)
		successTemplate.AddItem("✅", "状态", "已启用", msgelem.ItemTypeStatus)
		for key, value := range configPreview {
			successTemplate.AddItem("⚙️", key, value, msgelem.ItemTypeText)
		}
		successTemplate.AddAction("使用 /storage 设为默认存储")
		successTemplate.AddAction("使用 /storage_list 查看所有存储配置")

		// 使用格式化消息发送
		text, entities := successTemplate.BuildFormattedMessage()
		err := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if err != nil {
			ctx.Reply(update, ext.ReplyTextString(successTemplate.BuildMessage()), nil)
		}
	}

	// 清理缓存
	cache.Del(dataID)

	return dispatcher.EndGroups
}

// parseStorageConfig 解析存储配置
func parseStorageConfig(storageType, input string, expectedFields []string) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	switch storageType {
	case "alist":
		parts := strings.Split(strings.TrimSpace(input), ",")
		if len(parts) < 3 {
			return nil, fmt.Errorf("至少需要3个参数：URL,用户名,密码[,base_path]")
		}

		config["url"] = strings.TrimSpace(parts[0])
		config["username"] = strings.TrimSpace(parts[1])
		config["password"] = strings.TrimSpace(parts[2])

		if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
			config["base_path"] = strings.TrimSpace(parts[3])
		} else {
			config["base_path"] = "/"
		}

	case "webdav":
		parts := strings.Split(strings.TrimSpace(input), ",")
		if len(parts) < 3 {
			return nil, fmt.Errorf("至少需要3个参数：URL,用户名,密码[,路径]")
		}

		config["url"] = strings.TrimSpace(parts[0])
		config["username"] = strings.TrimSpace(parts[1])
		config["password"] = strings.TrimSpace(parts[2])

		if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
			config["path"] = strings.TrimSpace(parts[3])
		} else {
			config["path"] = "/"
		}

	case "minio":
		parts := strings.Split(strings.TrimSpace(input), ",")
		if len(parts) < 4 {
			return nil, fmt.Errorf("至少需要4个参数：endpoint,access_key,secret_key,bucket[,region]")
		}

		config["endpoint"] = strings.TrimSpace(parts[0])
		config["access_key"] = strings.TrimSpace(parts[1])
		config["secret_key"] = strings.TrimSpace(parts[2])
		config["bucket"] = strings.TrimSpace(parts[3])

		if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
			config["region"] = strings.TrimSpace(parts[4])
		} else {
			config["region"] = "us-east-1"
		}

	case "local":
		path := strings.TrimSpace(input)
		if path == "" {
			return nil, fmt.Errorf("路径不能为空")
		}
		config["base_path"] = path

	case "telegram":
		chatID := strings.TrimSpace(input)
		if chatID == "" {
			return nil, fmt.Errorf("chat_id不能为空")
		}

		// 验证chat_id格式（应该是数字，通常是负数）
		if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
			return nil, fmt.Errorf("chat_id必须是有效的整数")
		}
		config["chat_id"] = chatID

	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", storageType)
	}

	return config, nil
}

// findActiveStorageWizard 查找活跃的存储配置向导
func findActiveStorageWizard(chatID int64) (*tcbdata.StorageConfigWizard, string) {
	// 使用固定的缓存键格式
	key := fmt.Sprintf("storage_wizard_%d", chatID)
	if data, ok := cache.Get[tcbdata.StorageConfigWizard](key); ok {
		return &data, key
	}
	return nil, ""
}

// clearStorageWizardCache 清理存储向导缓存
func clearStorageWizardCache(chatID int64) {
	// 清理固定的向导缓存键
	key := fmt.Sprintf("storage_wizard_%d", chatID)
	cache.Del(key)
}

// handleDeleteStorageConfirmCallback 处理删除存储确认回调
func handleDeleteStorageConfirmCallback(ctx *ext.Context, update *ext.Update) error {
	dataParts := strings.Split(string(update.CallbackQuery.Data), " ")
	if len(dataParts) != 2 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的操作数据",
		})
		return dispatcher.EndGroups
	}

	dataID := dataParts[1]
	data, ok := cache.Get[tcbdata.DeleteStorageConfirm](dataID)
	if !ok {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "操作已过期",
		})
		return dispatcher.EndGroups
	}

	// 验证用户权限
	if data.ChatID != update.CallbackQuery.GetUserID() {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无权限执行此操作",
		})
		return dispatcher.EndGroups
	}

	// 执行删除
	if err := database.DeleteUserStorageByID(ctx, data.StorageID); err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "删除失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}

	// 更新消息
	ctx.EditMessage(data.ChatID, &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: "✅ 存储配置已删除",
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
		Message: "删除成功",
	})

	return dispatcher.EndGroups
}

// handleStorageToggleCallback 处理存储状态切换回调
func handleStorageToggleCallback(ctx *ext.Context, update *ext.Update) error {
	dataParts := strings.Split(string(update.CallbackQuery.Data), " ")
	if len(dataParts) != 2 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的操作数据",
		})
		return dispatcher.EndGroups
	}

	storageIDStr := dataParts[1]
	storageID, err := strconv.ParseUint(storageIDStr, 10, 32)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的存储ID",
		})
		return dispatcher.EndGroups
	}

	userID := update.CallbackQuery.GetUserID()
	if err := storage.Manager.ToggleUserStorageStatus(ctx, userID, uint(storageID)); err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "操作失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
		Message: "状态已切换",
	})

	// 可以选择刷新存储列表
	return handleStorageListCmd(ctx, update)
}

// handleStorageInfoCallback 处理存储信息显示回调
func handleStorageInfoCallback(ctx *ext.Context, update *ext.Update) error {
	// 解析存储ID
	callbackData := string(update.CallbackQuery.Data)
	parts := strings.Split(callbackData, " ")
	if len(parts) != 2 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的回调数据",
		})
		return dispatcher.EndGroups
	}

	storageID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的存储ID",
		})
		return dispatcher.EndGroups
	}

	// 获取存储配置
	storage, err := database.GetUserStorageByID(ctx, uint(storageID))
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "存储配置不存在",
		})
		return dispatcher.EndGroups
	}

	// 检查权限
	chatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil || storage.UserID != user.ID {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无权限查看此存储配置",
		})
		return dispatcher.EndGroups
	}

	// 使用模板系统构建存储信息
	template := msgelem.NewInfoTemplate("存储详情", storage.Name)

	// 添加基本信息
	template.AddItem("📦", "类型", strings.ToUpper(storage.Type), msgelem.ItemTypeText)

	status := "启用"
	statusIcon := "🟢"
	if !storage.Enable {
		status = "禁用"
		statusIcon = "🔴"
	}
	template.AddItem(statusIcon, "状态", status, msgelem.ItemTypeStatus)

	if storage.Description != "" {
		template.AddItem("📝", "描述", storage.Description, msgelem.ItemTypeText)
	}

	template.AddItem("🕐", "创建时间", storage.CreatedAt.Format("2006-01-02 15:04:05"), msgelem.ItemTypeText)
	template.AddItem("🔄", "更新时间", storage.UpdatedAt.Format("2006-01-02 15:04:05"), msgelem.ItemTypeText)

	// 解析并显示配置信息（脱敏）
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(storage.Config), &config); err == nil {
		for key, value := range config {
			if key == "password" || key == "secret_key" || key == "token" {
				template.AddItem("⚙️", key, "****", msgelem.ItemTypeText)
			} else {
				template.AddItem("⚙️", key, fmt.Sprintf("%v", value), msgelem.ItemTypeText)
			}
		}
	}

	// 创建操作按钮
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🧪 测试连接",
						Data: []byte(fmt.Sprintf("storage_test %d", storage.ID)),
					},
					&tg.KeyboardButtonCallback{
						Text: "✏️ 编辑配置",
						Data: []byte(fmt.Sprintf("storage_edit %d", storage.ID)),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🗑️ 删除存储",
						Data: []byte(fmt.Sprintf("storage_delete %d", storage.ID)),
					},
					&tg.KeyboardButtonCallback{
						Text: "🔙 返回列表",
						Data: []byte("storage_back_to_list"),
					},
				},
			},
		},
	}

	// 使用格式化消息编辑
	text, entities := template.BuildFormattedMessage()
	callback := update.CallbackQuery
	userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
	err = msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, markup)
	if err != nil {
		// 如果格式化编辑失败，回退到普通编辑
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          update.CallbackQuery.GetMsgID(),
			Message:     template.BuildMessage(),
			ReplyMarkup: markup,
		})
	}

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleStorageAddStartCallback 处理添加存储开始回调
func handleStorageAddStartCallback(ctx *ext.Context, update *ext.Update) error {
	// 显示存储类型选择
	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🌐 Alist",
						Data: []byte("storage_type_alist"),
					},
					&tg.KeyboardButtonCallback{
						Text: "📁 WebDAV",
						Data: []byte("storage_type_webdav"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "☁️ MinIO/S3",
						Data: []byte("storage_type_minio"),
					},
					&tg.KeyboardButtonCallback{
						Text: "💻 本地存储",
						Data: []byte("storage_type_local"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📱 Telegram",
						Data: []byte("storage_type_telegram"),
					},
					&tg.KeyboardButtonCallback{
						Text: "❌ 取消",
						Data: []byte("cancel"),
					},
				},
			},
		},
	}

	// 使用模板系统构建选择消息
	template := msgelem.NewInfoTemplate("选择存储类型", "请选择您要添加的存储类型")
	template.AddItem("🌐", "Alist", "支持多种网盘的聚合平台", msgelem.ItemTypeText)
	template.AddItem("📁", "WebDAV", "标准WebDAV协议存储", msgelem.ItemTypeText)
	template.AddItem("☁️", "MinIO/S3", "S3兼容对象存储", msgelem.ItemTypeText)
	template.AddItem("💻", "本地存储", "服务器本地磁盘", msgelem.ItemTypeText)
	template.AddItem("📱", "Telegram", "Telegram频道/群组存储", msgelem.ItemTypeText)

	// 使用格式化消息编辑
	text, entities := template.BuildFormattedMessage()
	callback := update.CallbackQuery
	userPeer := &tg.InputPeerUser{UserID: callback.Peer.(*tg.PeerUser).UserID}
	err := msgelem.EditWithFormattedText(ctx, userPeer, callback.MsgID, text, entities, markup)
	if err != nil {
		// 如果格式化编辑失败，回退到普通编辑
		ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
			ID:          update.CallbackQuery.GetMsgID(),
			Message:     template.BuildMessage(),
			ReplyMarkup: markup,
		})
	}

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleStorageBackToListCallback 处理返回存储列表回调
func handleStorageBackToListCallback(ctx *ext.Context, update *ext.Update) error {
	chatID := update.GetUserChat().GetID()

	var message strings.Builder
	message.WriteString("📚 存储配置列表:\n\n")

	// 获取系统配置的存储
	systemStorages := storage.GetUserStorages(ctx, chatID)
	if len(systemStorages) > 0 {
		message.WriteString("🏢 **系统配置存储**:\n")
		for _, stor := range systemStorages {
			message.WriteString(fmt.Sprintf("🟢 **%s** (%s)\n", stor.Name(), stor.Type()))
			message.WriteString("   📝 系统配置文件定义\n\n")
		}
	}

	// 获取用户自定义存储配置
	userStorages, err := database.GetUserStoragesByChatID(ctx, chatID)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "获取用户存储列表失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}

	var markup *tg.ReplyInlineMarkup
	if len(userStorages) > 0 {
		message.WriteString("👤 **用户自定义存储**:\n")
		for _, userStorage := range userStorages {
			status := "🟢"
			if !userStorage.Enable {
				status = "🔴"
			}

			message.WriteString(fmt.Sprintf("%s **%s** (%s)\n", status, userStorage.Name, userStorage.Type))
			if userStorage.Description != "" {
				message.WriteString(fmt.Sprintf("   📝 %s\n", userStorage.Description))
			}
			message.WriteString(fmt.Sprintf("   🕐 创建时间: %s\n\n", userStorage.CreatedAt.Format("2006-01-02 15:04:05")))
		}

		// 添加操作按钮（仅针对用户存储）
		markup, _ = msgelem.BuildStorageManageMarkup(ctx, userStorages)
	} else {
		if len(systemStorages) == 0 {
			message.WriteString("❌ 暂无可用的存储配置\n\n")
			message.WriteString("💡 使用 /storage_list 查看和添加存储配置")
		}
	}

	// 编辑消息
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     message.String(),
		ReplyMarkup: markup,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
		Message: "列表已更新",
	})

	return dispatcher.EndGroups
}

// handleStorageTypeCallback 处理存储类型选择回调
func handleStorageTypeCallback(ctx *ext.Context, update *ext.Update) error {
	callbackData := string(update.CallbackQuery.Data)
	parts := strings.Split(callbackData, "_")
	if len(parts) != 3 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的回调数据",
		})
		return dispatcher.EndGroups
	}

	storageType := parts[2] // storage_type_alist -> alist

	// 提示用户输入存储名称
	text := fmt.Sprintf(`🏷️ 输入存储名称

请为您的 %s 存储配置起一个名称：

示例: 我的%s, %s1, 备份%s

💡 请直接回复此消息，输入存储名称`,
		strings.ToUpper(storageType), storageType, storageType, storageType)

	// 保存存储类型到缓存，等待用户输入名称
	chatID := update.GetUserChat().GetID()
	cacheKey := fmt.Sprintf("storage_name_input_%d", chatID)
	cache.Set(cacheKey, storageType)

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: text,
		ReplyMarkup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "❌ 取消",
							Data: []byte("cancel"),
						},
					},
				},
			},
		},
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleStorageEditCallback 处理编辑配置回调
func handleStorageTestCallback(ctx *ext.Context, update *ext.Update) error {
	// 解析存储ID
	callbackData := string(update.CallbackQuery.Data)
	parts := strings.Split(callbackData, " ")
	if len(parts) != 2 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的操作数据",
		})
		return dispatcher.EndGroups
	}

	storageIDStr := parts[1]
	storageID, err := strconv.ParseUint(storageIDStr, 10, 32)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的存储ID",
		})
		return dispatcher.EndGroups
	}

	userID := update.CallbackQuery.GetUserID()

	// 获取存储信息
	userStorage, err := database.GetUserStorageByID(ctx, uint(storageID))
	if err != nil || userStorage == nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "存储配置不存在",
		})
		return dispatcher.EndGroups
	}

	// 验证用户权限
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil || user.ID != userStorage.UserID {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无权限访问此存储配置",
		})
		return dispatcher.EndGroups
	}

	// 测试存储连接
	err = storage.Manager.TestUserStorageConnection(ctx, userID, userStorage.Name)

	var message string
	if err != nil {
		message = fmt.Sprintf("❌ 存储 '%s' 连接测试失败: %s", userStorage.Name, err.Error())
	} else {
		message = fmt.Sprintf("✅ 存储 '%s' 连接测试成功", userStorage.Name)
	}

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
		Alert:   true,
		Message: message,
	})

	return dispatcher.EndGroups
}
func handleStorageEditCallback(ctx *ext.Context, update *ext.Update) error {
	// 解析存储ID
	callbackData := string(update.CallbackQuery.Data)
	parts := strings.Split(callbackData, " ")
	if len(parts) != 2 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的操作数据",
		})
		return dispatcher.EndGroups
	}

	storageIDStr := parts[1]
	storageID, err := strconv.ParseUint(storageIDStr, 10, 32)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的存储ID",
		})
		return dispatcher.EndGroups
	}

	userID := update.CallbackQuery.GetUserID()

	// 获取存储信息
	userStorage, err := database.GetUserStorageByID(ctx, uint(storageID))
	if err != nil || userStorage == nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "存储配置不存在",
		})
		return dispatcher.EndGroups
	}

	// 验证用户权限
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil || user.ID != userStorage.UserID {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无权限访问此存储配置",
		})
		return dispatcher.EndGroups
	}

	// 显示当前配置并开始编辑向导
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(userStorage.Config), &configData); err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "解析存储配置失败",
		})
		return dispatcher.EndGroups
	}

	configText := "📝 当前配置:\n\n"
	for key, value := range configData {
		if key == "password" || key == "secret_key" {
			configText += fmt.Sprintf("• %s: %s\n", key, "******")
		} else {
			configText += fmt.Sprintf("• %s: %v\n", key, value)
		}
	}

	configText += "\n🔧 请发送新的配置信息 (格式与添加时相同)"

	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:      update.CallbackQuery.GetMsgID(),
		Message: configText,
		ReplyMarkup: &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "❌ 取消",
							Data: []byte("cancel"),
						},
					},
				},
			},
		},
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
	})

	// 开始配置编辑向导
	return startStorageConfigWizard(ctx, update, userStorage.Name, userStorage.Type, userStorage.Description)
}

// handleStorageDeleteCallback 处理删除存储回调
func handleStorageDeleteCallback(ctx *ext.Context, update *ext.Update) error {
	// 解析存储ID
	callbackData := string(update.CallbackQuery.Data)
	parts := strings.Split(callbackData, " ")
	if len(parts) != 2 {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的操作数据",
		})
		return dispatcher.EndGroups
	}

	storageIDStr := parts[1]
	storageID, err := strconv.ParseUint(storageIDStr, 10, 32)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无效的存储ID",
		})
		return dispatcher.EndGroups
	}

	userID := update.CallbackQuery.GetUserID()

	// 获取存储信息
	userStorage, err := database.GetUserStorageByID(ctx, uint(storageID))
	if err != nil || userStorage == nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "存储配置不存在",
		})
		return dispatcher.EndGroups
	}

	// 验证用户权限
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil || user.ID != userStorage.UserID {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "无权限访问此存储配置",
		})
		return dispatcher.EndGroups
	}

	// 检查是否为默认存储
	if user.DefaultStorage == userStorage.Name {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.CallbackQuery.GetQueryID(),
			Alert:   true,
			Message: "❌ 无法删除默认存储，请先设置其他存储为默认",
		})
		return dispatcher.EndGroups
	}

	// 创建确认按钮
	confirmData := tcbdata.DeleteStorageConfirm{
		StorageID: userStorage.ID,
		ChatID:    userID,
	}
	dataID := fmt.Sprintf("confirm_delete_%d_%d", confirmData.ChatID, confirmData.StorageID)
	cache.Set(dataID, confirmData)

	markup := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🗑️ 确认删除",
						Data: []byte(tcbdata.TypeDeleteStorageConfirm + " " + dataID),
					},
					&tg.KeyboardButtonCallback{
						Text: "❌ 取消",
						Data: []byte("cancel"),
					},
				},
			},
		},
	}

	confirmText := fmt.Sprintf("⚠️ 确认删除存储配置 '%s'?\n\n此操作不可撤销！", userStorage.Name)
	ctx.EditMessage(update.EffectiveChat().GetID(), &tg.MessagesEditMessageRequest{
		ID:          update.CallbackQuery.GetMsgID(),
		Message:     confirmText,
		ReplyMarkup: markup,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.CallbackQuery.GetQueryID(),
	})

	return dispatcher.EndGroups
}
