package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/database"
	ruletemplate "github.com/krau/SaveAny-Bot/pkg/rule"
	"github.com/krau/SaveAny-Bot/storage"
)

// 规则向导状态管理
type RuleWizardState struct {
	UserID      int64
	Step        int
	RuleType    string
	Template    string
	Data        string
	StorageName string
	DirPath     string
	MessageID   int
}

// 全局向导状态存储 (在生产环境中应该使用 Redis 或数据库)
var ruleWizardStates = make(map[int64]*RuleWizardState)

// 用户输入状态管理
type UserInputState struct {
	UserID      int64
	Type        string // "rule_data", "rule_path"
	Step        int
	RuleWizard  *RuleWizardState
}

var userInputStates = make(map[int64]*UserInputState)

// handleRuleCallback 处理规则相关的内联键盘回调
func handleRuleCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	data := string(callback.Data)
	
	log.Infof("=== handleRuleCallback 被调用，数据=%s ===", data)

	switch {
	case strings.HasPrefix(data, "rule_list"):
		return handleRuleListCallback(ctx, update)
	case strings.HasPrefix(data, "rule_switch"):
		return handleRuleSwitchCallback(ctx, update)
	case strings.HasPrefix(data, "rule_add_start"):
		return handleRuleAddStartCallback(ctx, update)
	case strings.HasPrefix(data, "rule_add_type_"):
		return handleRuleAddTypeCallback(ctx, update, data)
	case strings.HasPrefix(data, "rule_add_template_"):
		return handleRuleAddTemplateCallback(ctx, update, data)
	case strings.HasPrefix(data, "rule_wizard_"):
		return handleRuleWizardCallback(ctx, update, data)
	case strings.HasPrefix(data, "rule_view_list"):
		return handleRuleViewListCallback(ctx, update)
	case strings.HasPrefix(data, "rule_delete_"):
		return handleRuleDeleteCallback(ctx, update, data)
	case strings.HasPrefix(data, "rule_detail_"):
		return handleRuleDetailCallback(ctx, update, data)
	case strings.HasPrefix(data, "rule_help"):
		return handleRuleHelpCallback(ctx, update)
	case strings.HasPrefix(data, "rule_back"):
		return handleRuleBackCallback(ctx, update)
	}

	// 回复确认
	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleListCallback 处理显示规则列表的回调
func handleRuleListCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "获取用户信息失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	template := msgelem.NewInfoTemplate("📋 规则管理", "管理您的自动文件组织规则")
	
	// 显示规则模式状态
	ruleStatusText := "已禁用"
	if user.ApplyRule {
		ruleStatusText = "已启用"
	}
	template.AddItem("⚙️", "规则模式", ruleStatusText, msgelem.ItemTypeStatus)
	template.AddItem("📊", "规则数量", fmt.Sprintf("%d", len(user.Rules)), msgelem.ItemTypeText)

	// 构建内联键盘
	keyboard := buildRuleManagementKeyboard(user.ApplyRule, len(user.Rules))

	messageText, entities := template.BuildFormattedMessage()

	// 编辑消息
	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleSwitchCallback 处理规则模式开关的回调
func handleRuleSwitchCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "获取用户信息失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	// 切换规则模式
	newApplyRule := !user.ApplyRule
	if err := database.UpdateUserApplyRule(ctx, user.ChatID, newApplyRule); err != nil {
		log.Errorf("更新用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "更新设置失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	// 刷新显示
	return handleRuleListCallback(ctx, update)
}

// handleRuleAddStartCallback 处理开始添加规则的回调
func handleRuleAddStartCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	// 初始化向导状态
	ruleWizardStates[userChatID] = &RuleWizardState{
		UserID:    userChatID,
		Step:      1,
		MessageID: callback.MsgID,
	}

	// 显示规则类型选择界面
	return showRuleTypeSelection(ctx, update)
}

// showRuleTypeSelection 显示规则类型选择界面
func showRuleTypeSelection(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery

	template := msgelem.NewInfoTemplate("📝 添加规则 - 步骤 1/4", "请选择规则类型")
	
	template.AddAction("选择规则类型以继续创建规则")

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📄 文件名匹配",
						Data: []byte("rule_add_type_FILENAME-REGEX"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "💬 消息内容匹配",
						Data: []byte("rule_add_type_MESSAGE-REGEX"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🖼️ 相册文件",
						Data: []byte("rule_add_type_IS-ALBUM"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回",
						Data: []byte("rule_back"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleAddTypeCallback 处理规则类型选择的回调
func handleRuleAddTypeCallback(ctx *ext.Context, update *ext.Update, data string) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	// 解析规则类型
	ruleType := strings.TrimPrefix(data, "rule_add_type_")
	
	// 更新向导状态
	state, exists := ruleWizardStates[userChatID]
	if !exists {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "会话已过期，请重新开始",
			Alert:   true,
		})
		return handleRuleListCallback(ctx, update)
	}

	state.RuleType = ruleType
	state.Step = 2

	// 显示模板选择界面
	return showRuleTemplateSelection(ctx, update, ruleType)
}

// showRuleTemplateSelection 显示规则模板选择界面
func showRuleTemplateSelection(ctx *ext.Context, update *ext.Update, ruleType string) error {
	callback := update.CallbackQuery

	template := msgelem.NewInfoTemplate("📝 添加规则 - 步骤 2/4", "选择规则模板或自定义")

	// 根据规则类型显示不同的模板
	templates := getRuleTemplates(ruleType)
	
	var keyboard *tg.ReplyInlineMarkup
	
	if len(templates) > 0 {
		// 有预定义模板时的界面
		template.AddAction("选择预定义模板快速创建，或选择自定义手动设置")
		
		var rows []tg.KeyboardButtonRow
		
		// 添加模板按钮
		for _, tmpl := range templates {
			rows = append(rows, tg.KeyboardButtonRow{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: tmpl.Icon + " " + tmpl.Name,
						Data: []byte("rule_add_template_" + tmpl.ID),
					},
				},
			})
		}
		
		// 添加自定义选项
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: "✏️ 自定义规则",
					Data: []byte("rule_add_template_custom"),
				},
			},
		})
		
		// 添加返回按钮
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: "⬅️ 返回",
					Data: []byte("rule_add_start"),
				},
			},
		})
		
		keyboard = &tg.ReplyInlineMarkup{Rows: rows}
	} else {
		// 没有预定义模板时直接进入自定义
		template.AddAction("该规则类型需要自定义设置")
		
		keyboard = &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "✏️ 开始自定义",
							Data: []byte("rule_add_template_custom"),
						},
					},
				},
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "⬅️ 返回",
							Data: []byte("rule_add_start"),
						},
					},
				},
			},
		}
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// buildRuleManagementKeyboard 构建规则管理界面的内联键盘
func buildRuleManagementKeyboard(applyRule bool, ruleCount int) *tg.ReplyInlineMarkup {
	var rows []tg.KeyboardButtonRow

	// 规则模式开关
	switchText := "🔴 启用规则模式"
	if applyRule {
		switchText = "🟢 禁用规则模式"
	}
	
	rows = append(rows, tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCallback{
				Text: switchText,
				Data: []byte("rule_switch"),
			},
		},
	})

	// 添加规则按钮
	rows = append(rows, tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCallback{
				Text: "➕ 添加规则",
				Data: []byte("rule_add_start"),
			},
		},
	})

	// 如果有规则，显示管理按钮
	if ruleCount > 0 {
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: "📋 查看规则列表",
					Data: []byte("rule_view_list"),
				},
			},
		})
	}

	// 帮助按钮
	rows = append(rows, tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCallback{
				Text: "❓ 帮助",
				Data: []byte("rule_help"),
			},
		},
	})

	return &tg.ReplyInlineMarkup{Rows: rows}
}

// handleRuleBackCallback 处理返回按钮的回调
func handleRuleBackCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	// 清理向导状态和输入状态
	delete(ruleWizardStates, userChatID)
	delete(userInputStates, userChatID)

	// 返回主界面
	return handleRuleListCallback(ctx, update)
}

// handleRuleViewListCallback 处理查看规则列表的回调
func handleRuleViewListCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "获取用户信息失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	template := msgelem.NewInfoTemplate("📋 规则列表", fmt.Sprintf("共 %d 条规则", len(user.Rules)))
	
	if len(user.Rules) == 0 {
		template.AddAction("暂无规则，点击添加规则创建您的第一条规则")
	} else {
		// 显示所有规则
		for _, rule := range user.Rules {
			ruleTypeIcon := getRuleTypeIcon(rule.Type)
			ruleDesc := fmt.Sprintf("%s → %s/%s", rule.Data, rule.StorageName, rule.DirPath)
			if len(ruleDesc) > 30 {
				ruleDesc = ruleDesc[:27] + "..."
			}
			template.AddItem(ruleTypeIcon, fmt.Sprintf("规则 %d", rule.ID), ruleDesc, msgelem.ItemTypeText)
		}
		template.AddAction("选择规则查看详情或删除")
	}

	// 构建内联键盘
	var keyboard *tg.ReplyInlineMarkup
	if len(user.Rules) > 0 {
		var rows []tg.KeyboardButtonRow
		
		// 添加规则操作按钮 (每行2个)
		for i := 0; i < len(user.Rules); i += 2 {
			var buttons []tg.KeyboardButtonClass
			
			// 第一个按钮
			rule1 := user.Rules[i]
			buttons = append(buttons, &tg.KeyboardButtonCallback{
				Text: fmt.Sprintf("📝 规则 %d", rule1.ID),
				Data: []byte(fmt.Sprintf("rule_detail_%d", rule1.ID)),
			})
			
			// 第二个按钮（如果存在）
			if i+1 < len(user.Rules) {
				rule2 := user.Rules[i+1]
				buttons = append(buttons, &tg.KeyboardButtonCallback{
					Text: fmt.Sprintf("📝 规则 %d", rule2.ID),
					Data: []byte(fmt.Sprintf("rule_detail_%d", rule2.ID)),
				})
			}
			
			rows = append(rows, tg.KeyboardButtonRow{Buttons: buttons})
		}
		
		// 添加控制按钮
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: "➕ 添加规则",
					Data: []byte("rule_add_start"),
				},
			},
		})
		
		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: "⬅️ 返回",
					Data: []byte("rule_back"),
				},
			},
		})
		
		keyboard = &tg.ReplyInlineMarkup{Rows: rows}
	} else {
		keyboard = &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "➕ 添加规则",
							Data: []byte("rule_add_start"),
						},
					},
				},
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "⬅️ 返回",
							Data: []byte("rule_back"),
						},
					},
				},
			},
		}
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleHelpCallback 处理帮助回调
func handleRuleHelpCallback(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery

	template := msgelem.NewInfoTemplate("❓ 规则帮助", "自动文件组织规则说明")
	
	template.AddItem("📄", "文件名匹配", "根据文件名正则表达式匹配", msgelem.ItemTypeText)
	template.AddItem("💬", "消息内容匹配", "根据消息内容正则表达式匹配", msgelem.ItemTypeText)
	template.AddItem("🖼️", "相册文件", "自动匹配相册中的所有文件", msgelem.ItemTypeText)

	template.AddAction("规则按优先级从上到下匹配，匹配成功后停止")
	template.AddAction("支持正则表达式，提供多种预设模板")

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回",
						Data: []byte("rule_back"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleDetailCallback 处理规则详情查看的回调
func handleRuleDetailCallback(ctx *ext.Context, update *ext.Update, data string) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	// 解析规则ID
	ruleIDStr := strings.TrimPrefix(data, "rule_detail_")
	ruleID, err := strconv.Atoi(ruleIDStr)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "无效的规则ID",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "获取用户信息失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	// 查找规则
	var targetRule *database.Rule
	for _, rule := range user.Rules {
		if rule.ID == uint(ruleID) {
			targetRule = &rule
			break
		}
	}

	if targetRule == nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "规则不存在",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	template := msgelem.NewInfoTemplate("📝 规则详情", fmt.Sprintf("规则 ID: %d", targetRule.ID))
	
	template.AddItem(getRuleTypeIcon(targetRule.Type), "规则类型", getRuleTypeName(targetRule.Type), msgelem.ItemTypeText)
	template.AddItem("🔍", "匹配条件", targetRule.Data, msgelem.ItemTypeCode)
	template.AddItem("📁", "存储位置", targetRule.StorageName, msgelem.ItemTypeText)
	template.AddItem("📂", "目录路径", targetRule.DirPath, msgelem.ItemTypeCode)
	template.AddItem("🕐", "创建时间", targetRule.CreatedAt.Format("2006-01-02 15:04"), msgelem.ItemTypeText)

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🗑️ 删除规则",
						Data: []byte(fmt.Sprintf("rule_delete_%d", targetRule.ID)),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回列表",
						Data: []byte("rule_view_list"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleDeleteCallback 处理规则删除的回调
func handleRuleDeleteCallback(ctx *ext.Context, update *ext.Update, data string) error {
	callback := update.CallbackQuery

	// 解析规则ID
	ruleIDStr := strings.TrimPrefix(data, "rule_delete_")
	ruleID, err := strconv.Atoi(ruleIDStr)
	if err != nil {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "无效的规则ID",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	// 删除规则
	if err := database.DeleteRule(ctx, uint(ruleID)); err != nil {
		log.Errorf("删除规则失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "删除规则失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
		Message: "规则删除成功",
	})

	// 返回规则列表
	return handleRuleViewListCallback(ctx, update)
}

// getRuleTypeIcon 获取规则类型图标
func getRuleTypeIcon(ruleType string) string {
	switch ruleType {
	case "FILENAME-REGEX":
		return "📄"
	case "MESSAGE-REGEX":
		return "💬"
	case "IS-ALBUM":
		return "🖼️"
	default:
		return "📋"
	}
}

// getRuleTypeName 获取规则类型名称
func getRuleTypeName(ruleType string) string {
	switch ruleType {
	case "FILENAME-REGEX":
		return "文件名匹配"
	case "MESSAGE-REGEX":
		return "消息内容匹配"
	case "IS-ALBUM":
		return "相册文件"
	default:
		return "未知类型"
	}
}

// handleRuleAddTemplateCallback 处理模板选择的回调
func handleRuleAddTemplateCallback(ctx *ext.Context, update *ext.Update, data string) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	// 解析模板ID
	templateID := strings.TrimPrefix(data, "rule_add_template_")
	
	// 获取向导状态
	state, exists := ruleWizardStates[userChatID]
	if !exists {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "会话已过期，请重新开始",
			Alert:   true,
		})
		return handleRuleListCallback(ctx, update)
	}

	if templateID == "custom" {
		// 自定义规则流程
		state.Template = "custom"
		state.Step = 3
		return showRuleDataInput(ctx, update)
	} else {
		// 使用模板
		template := ruletemplate.GetTemplateByID(templateID)
		if template == nil {
			ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
				QueryID: callback.GetQueryID(),
				Message: "模板不存在",
				Alert:   true,
			})
			return dispatcher.EndGroups
		}

		state.Template = templateID
		state.Data = template.DataPattern
		state.Step = 3
		return showRuleStorageSelection(ctx, update)
	}
}

// showRuleDataInput 显示规则数据输入界面（自定义规则）
func showRuleDataInput(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	state := ruleWizardStates[userChatID]
	
	template := msgelem.NewInfoTemplate("📝 添加规则 - 步骤 3/4", "请输入匹配条件")
	
	switch state.RuleType {
	case "FILENAME-REGEX":
		template.AddAction("请发送文件名的正则表达式")
		template.AddAction("示例：\\.(jpg|png|gif)$ （匹配图片文件）")
		template.AddAction("发送消息内容将被用作匹配条件")
	case "MESSAGE-REGEX":
		template.AddAction("请发送消息内容的正则表达式")
		template.AddAction("示例：(?i)(重要|urgent) （匹配重要消息）")
		template.AddAction("发送消息内容将被用作匹配条件")
	case "IS-ALBUM":
		template.AddAction("相册规则自动匹配，无需输入条件")
		// 相册规则直接设置数据
		state.Data = "true"
		return showRuleStorageSelection(ctx, update)
	}

	// 设置用户输入状态
	userInputStates[userChatID] = &UserInputState{
		UserID:     userChatID,
		Type:       "rule_data",
		Step:       3,
		RuleWizard: state,
	}

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回",
						Data: []byte("rule_add_type_" + state.RuleType),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// showRuleStorageSelection 显示存储选择界面
func showRuleStorageSelection(ctx *ext.Context, update *ext.Update) error {
	log.Infof("=== showRuleStorageSelection 函数被调用 ===")
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	log.Infof("规则向导：回调用户ID=%d", userChatID)

	// 获取用户的存储列表
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "获取用户信息失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	log.Infof("规则向导：获取到用户 ID=%d, ChatID=%d", user.ID, user.ChatID)

	// 获取用户的启用存储配置（数据库中的个人存储）
	userStorages, err := database.GetEnabledUserStoragesByUserID(ctx, user.ID)
	if err != nil {
		log.Errorf("获取用户启用存储失败: %s", err)
		userStorages = []database.UserStorage{} // 使用空列表作为fallback
	}

	log.Infof("规则向导：获取到 %d 个用户个人存储", len(userStorages))

	// 获取全局可用存储（配置文件中的存储）
	log.Infof("规则向导：调用 storage.GetUserStorages(ctx, %d)", userChatID)
	
	// 先检查 UserStorages 缓存
	log.Infof("规则向导：检查 storage.UserStorages 缓存...")
	
	// 直接调用配置检查函数
	log.Infof("规则向导：调用 config.Cfg.GetStorageNamesByUserID(%d)", userChatID)
	configNames := config.Cfg.GetStorageNamesByUserID(userChatID)
	log.Infof("规则向导：config返回的存储名列表: %v", configNames)
	
	// 检查配置中是否有该用户
	log.Infof("规则向导：检查用户是否有存储权限...")
	hasStorage1 := config.Cfg.HasStorage(userChatID, "本机1")
	hasStorage2 := config.Cfg.HasStorage(userChatID, "openlist")
	log.Infof("规则向导：HasStorage(本机1)=%t, HasStorage(openlist)=%t", hasStorage1, hasStorage2)
	
	globalStorages := storage.GetUserStorages(ctx, userChatID)
	log.Infof("规则向导：获取到 %d 个全局可用存储", len(globalStorages))
	for i, globalStorage := range globalStorages {
		log.Infof("  全局存储 %d: Name=%s, Type=%s", i, globalStorage.Name(), globalStorage.Type())
	}

	// 合并存储列表
	type AvailableStorage struct {
		Name string
		Type string
		Icon string
	}
	
	var availableStorages []AvailableStorage
	storageMap := make(map[string]bool) // 防重复

	// 添加用户个人存储
	for _, userStorage := range userStorages {
		if !storageMap[userStorage.Name] {
			storageMap[userStorage.Name] = true
			availableStorages = append(availableStorages, AvailableStorage{
				Name: userStorage.Name,
				Type: userStorage.Type,
			})
		}
	}

	// 添加全局存储（排除重复）
	for _, globalStorage := range globalStorages {
		storageName := globalStorage.Name()
		if !storageMap[storageName] {
			storageMap[storageName] = true
			availableStorages = append(availableStorages, AvailableStorage{
				Name: storageName,
				Type: string(globalStorage.Type()),
			})
		}
	}

	log.Infof("规则向导：合并后共有 %d 个可用存储", len(availableStorages))

	template := msgelem.NewInfoTemplate("📝 添加规则 - 步骤 3/4", "选择存储位置")
	
	if len(availableStorages) == 0 {
		log.Warnf("规则向导：没有可用存储，显示错误消息")
		
		template.AddAction("您还没有配置任何存储，请先配置存储")
		template.AddAction("使用 /storage 命令添加个人存储配置")

		keyboard := &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "➕ 添加存储",
							Data: []byte("storage_add_start"),
						},
					},
				},
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "⬅️ 返回",
							Data: []byte("rule_back"),
						},
					},
				},
			},
		}

		messageText, entities := template.BuildFormattedMessage()

		ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
			ID:        callback.MsgID,
			Message:   messageText,
			Entities:  entities,
			ReplyMarkup: keyboard,
		})

		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
		})

		return dispatcher.EndGroups
	}

	log.Infof("规则向导：显示 %d 个可用存储", len(availableStorages))

	template.AddItem("📊", "可用存储", fmt.Sprintf("%d 个", len(availableStorages)), msgelem.ItemTypeText)
	template.AddAction("选择要保存文件的存储位置")

	// 构建存储选择键盘
	var rows []tg.KeyboardButtonRow

	for _, storage := range availableStorages {
		// 根据存储类型设置图标
		storageIcon := "📁"
		switch storage.Type {
		case "local":
			storageIcon = "💻"
		case "alist":
			storageIcon = "🌐"
		case "webdav":
			storageIcon = "☁️"
		case "minio":
			storageIcon = "🗄️"
		case "telegram":
			storageIcon = "📱"
		}

		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: fmt.Sprintf("%s %s", storageIcon, storage.Name),
					Data: []byte(fmt.Sprintf("rule_wizard_storage_%s", storage.Name)),
				},
			},
		})
	}

	// 添加返回按钮
	rows = append(rows, tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCallback{
				Text: "⬅️ 返回",
				Data: []byte("rule_add_template_" + ruleWizardStates[userChatID].Template),
			},
		},
	})

	keyboard := &tg.ReplyInlineMarkup{Rows: rows}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleWizardCallback 处理向导步骤的回调
func handleRuleWizardCallback(ctx *ext.Context, update *ext.Update, data string) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	// 获取向导状态
	state, exists := ruleWizardStates[userChatID]
	if !exists {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "会话已过期，请重新开始",
			Alert:   true,
		})
		return handleRuleListCallback(ctx, update)
	}

	if strings.HasPrefix(data, "rule_wizard_storage_") {
		// 处理存储选择
		storageName := strings.TrimPrefix(data, "rule_wizard_storage_")
		state.StorageName = storageName
		state.Step = 4
		return showRulePathInput(ctx, update)
	} else if strings.HasPrefix(data, "rule_wizard_path_root") {
		// 处理根目录路径选择
		state.DirPath = ""
		delete(userInputStates, userChatID) // 清理输入状态
		return showRuleConfirm(ctx, update)
	} else if strings.HasPrefix(data, "rule_wizard_path_preset_") {
		// 处理预设路径选择
		presetPath := strings.TrimPrefix(data, "rule_wizard_path_preset_")
		state.DirPath = presetPath
		delete(userInputStates, userChatID) // 清理输入状态
		return showRuleConfirm(ctx, update)
	} else if strings.HasPrefix(data, "rule_wizard_confirm") {
		// 处理规则确认创建
		return handleRuleCreate(ctx, update)
	} else if strings.HasPrefix(data, "rule_wizard_back_to_storage") {
		// 返回存储选择
		state.Step = 3
		delete(userInputStates, userChatID) // 清理输入状态
		return showRuleStorageSelection(ctx, update)
	}

	return dispatcher.EndGroups
}

// handleRuleInputMessage 处理规则输入的文本消息
func handleRuleInputMessage(ctx *ext.Context, update *ext.Update) error {
	userChatID := update.GetUserChat().GetID()
	messageText := update.EffectiveMessage.Text

	// 检查是否有输入状态
	inputState, exists := userInputStates[userChatID]
	if !exists {
		return dispatcher.EndGroups // 不是规则输入状态，继续其他处理
	}

	switch inputState.Type {
	case "rule_data":
		// 处理规则数据输入
		if messageText == "" {
			ctx.Reply(update, ext.ReplyTextString("请输入有效的匹配条件"), nil)
			return dispatcher.EndGroups
		}

		// 更新向导状态
		inputState.RuleWizard.Data = messageText
		inputState.RuleWizard.Step = 4

		// 清理输入状态
		delete(userInputStates, userChatID)

		// 发送确认消息并继续流程
		ctx.Reply(update, ext.ReplyTextString("已接收匹配条件: "+messageText+"\n\n现在请选择存储位置"), nil)

		// 创建一个虚拟的 update 来继续向导流程
		return showRuleStorageSelectionFromInput(ctx, userChatID, inputState.RuleWizard)

	case "rule_path":
		// 处理路径输入
		if messageText == "" {
			messageText = "" // 空路径表示根目录
		}

		// 更新向导状态
		inputState.RuleWizard.DirPath = messageText

		// 清理输入状态
		delete(userInputStates, userChatID)

		// 发送确认消息并继续流程
		dirDisplay := messageText
		if dirDisplay == "" {
			dirDisplay = "根目录"
		}
		ctx.Reply(update, ext.ReplyTextString("已设置路径: "+dirDisplay+"\n\n请确认规则信息"), nil)

		// 显示确认界面
		return showRuleConfirmFromInput(ctx, userChatID, inputState.RuleWizard)

	default:
		// 清理无效状态
		delete(userInputStates, userChatID)
		return dispatcher.EndGroups
	}
}

// showRuleStorageSelectionFromInput 从输入处理显示存储选择界面
func showRuleStorageSelectionFromInput(ctx *ext.Context, userChatID int64, state *RuleWizardState) error {
	// 获取用户的存储列表
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.SendMessage(userChatID, &tg.MessagesSendMessageRequest{
			Message: "获取用户信息失败",
		})
		return dispatcher.EndGroups
	}

	// 获取用户的启用存储配置（数据库中的个人存储）
	userStorages, err := database.GetEnabledUserStoragesByUserID(ctx, user.ID)
	if err != nil {
		log.Errorf("获取用户启用存储失败: %s", err)
		userStorages = []database.UserStorage{} // 使用空列表作为fallback
	}

	// 获取全局可用存储（配置文件中的存储）
	globalStorages := storage.GetUserStorages(ctx, userChatID)

	// 合并存储列表
	type AvailableStorage struct {
		Name string
		Type string
	}
	
	var availableStorages []AvailableStorage
	storageMap := make(map[string]bool) // 防重复

	// 添加用户个人存储
	for _, userStorage := range userStorages {
		if !storageMap[userStorage.Name] {
			storageMap[userStorage.Name] = true
			availableStorages = append(availableStorages, AvailableStorage{
				Name: userStorage.Name,
				Type: userStorage.Type,
			})
		}
	}

	// 添加全局存储（排除重复）
	for _, globalStorage := range globalStorages {
		storageName := globalStorage.Name()
		if !storageMap[storageName] {
			storageMap[storageName] = true
			availableStorages = append(availableStorages, AvailableStorage{
				Name: storageName,
				Type: string(globalStorage.Type()),
			})
		}
	}

	template := msgelem.NewInfoTemplate("📝 添加规则 - 步骤 3/4", "选择存储位置")
	
	if len(availableStorages) == 0 {
		template.AddAction("您还没有配置任何存储，请先配置存储")
		template.AddAction("使用 /storage 命令添加个人存储配置")

		keyboard := &tg.ReplyInlineMarkup{
			Rows: []tg.KeyboardButtonRow{
				{
					Buttons: []tg.KeyboardButtonClass{
						&tg.KeyboardButtonCallback{
							Text: "➕ 添加存储",
							Data: []byte("storage_add_start"),
						},
					},
				},
			},
		}

		messageText, entities := template.BuildFormattedMessage()
		err = msgelem.SendFormattedMessage(ctx, userChatID, messageText, entities, keyboard)
		return err
	}

	template.AddItem("📊", "可用存储", fmt.Sprintf("%d 个", len(availableStorages)), msgelem.ItemTypeText)
	template.AddAction("选择要保存文件的存储位置")

	// 构建存储选择键盘
	var rows []tg.KeyboardButtonRow

	for _, storage := range availableStorages {
		// 根据存储类型设置图标
		storageIcon := "📁"
		switch storage.Type {
		case "local":
			storageIcon = "💻"
		case "alist":
			storageIcon = "🌐"
		case "webdav":
			storageIcon = "☁️"
		case "minio":
			storageIcon = "🗄️"
		case "telegram":
			storageIcon = "📱"
		}

		rows = append(rows, tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: fmt.Sprintf("%s %s", storageIcon, storage.Name),
					Data: []byte(fmt.Sprintf("rule_wizard_storage_%s", storage.Name)),
				},
			},
		})
	}

	keyboard := &tg.ReplyInlineMarkup{Rows: rows}
	messageText, entities := template.BuildFormattedMessage()

	err = msgelem.SendFormattedMessage(ctx, userChatID, messageText, entities, keyboard)
	return err
}

// showRuleConfirmFromInput 从输入处理显示确认界面
func showRuleConfirmFromInput(ctx *ext.Context, userChatID int64, state *RuleWizardState) error {
	template := msgelem.NewInfoTemplate("📝 确认规则", "请确认规则信息并创建")
	
	// 显示规则摘要
	template.AddItem(getRuleTypeIcon(state.RuleType), "规则类型", getRuleTypeName(state.RuleType), msgelem.ItemTypeText)
	template.AddItem("🔍", "匹配条件", state.Data, msgelem.ItemTypeCode)
	template.AddItem("📁", "存储位置", state.StorageName, msgelem.ItemTypeText)
	
	dirPath := state.DirPath
	if dirPath == "" {
		dirPath = "根目录"
	}
	template.AddItem("📂", "目录路径", dirPath, msgelem.ItemTypeCode)

	template.AddAction("确认信息无误后点击创建规则")

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "✅ 创建规则",
						Data: []byte("rule_wizard_confirm"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "❌ 取消",
						Data: []byte("rule_back"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()
	err := msgelem.SendFormattedMessage(ctx, userChatID, messageText, entities, keyboard)
	return err
}

// showRuleConfirm 显示规则确认界面
func showRuleConfirm(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	state := ruleWizardStates[userChatID]
	
	template := msgelem.NewInfoTemplate("📝 确认规则", "请确认规则信息并创建")
	
	// 显示规则摘要
	template.AddItem(getRuleTypeIcon(state.RuleType), "规则类型", getRuleTypeName(state.RuleType), msgelem.ItemTypeText)
	template.AddItem("🔍", "匹配条件", state.Data, msgelem.ItemTypeCode)
	template.AddItem("📁", "存储位置", state.StorageName, msgelem.ItemTypeText)
	
	dirPath := state.DirPath
	if dirPath == "" {
		dirPath = "根目录"
	}
	template.AddItem("📂", "目录路径", dirPath, msgelem.ItemTypeCode)

	template.AddAction("确认信息无误后点击创建规则")

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "✅ 创建规则",
						Data: []byte("rule_wizard_confirm"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回修改",
						Data: []byte("rule_wizard_back_to_storage"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// showRulePathInput 显示路径输入界面
func showRulePathInput(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	state := ruleWizardStates[userChatID]
	
	template := msgelem.NewInfoTemplate("📝 添加规则 - 步骤 4/4", "设置目录路径")
	
	template.AddItem("📁", "存储位置", state.StorageName, msgelem.ItemTypeText)
	template.AddAction("选择快捷路径或发送自定义路径")
	template.AddAction("示例：/downloads/images 或 images/")
	template.AddAction("留空表示保存到根目录")

	// 设置用户输入状态
	userInputStates[userChatID] = &UserInputState{
		UserID:     userChatID,
		Type:       "rule_path",
		Step:       4,
		RuleWizard: state,
	}

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📁 根目录",
						Data: []byte("rule_wizard_path_root"),
					},
					&tg.KeyboardButtonCallback{
						Text: "📂 downloads",
						Data: []byte("rule_wizard_path_preset_downloads"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🖼️ images",
						Data: []byte("rule_wizard_path_preset_images"),
					},
					&tg.KeyboardButtonCallback{
						Text: "🎥 videos",
						Data: []byte("rule_wizard_path_preset_videos"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📄 documents",
						Data: []byte("rule_wizard_path_preset_documents"),
					},
					&tg.KeyboardButtonCallback{
						Text: "🎵 music",
						Data: []byte("rule_wizard_path_preset_music"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回",
						Data: []byte("rule_wizard_back_to_storage"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
	})

	return dispatcher.EndGroups
}

// handleRuleCreate 处理规则创建
func handleRuleCreate(ctx *ext.Context, update *ext.Update) error {
	callback := update.CallbackQuery
	userChatID := callback.GetUserID()

	state, exists := ruleWizardStates[userChatID]
	if !exists {
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "会话已过期，请重新开始",
			Alert:   true,
		})
		return handleRuleListCallback(ctx, update)
	}

	// 获取用户信息
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		log.Errorf("获取用户失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "获取用户信息失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	// 创建规则
	newRule := &database.Rule{
		Type:        state.RuleType,
		Data:        state.Data,
		StorageName: state.StorageName,
		DirPath:     state.DirPath,
		UserID:      user.ID,
	}

	if err := database.CreateRule(ctx, newRule); err != nil {
		log.Errorf("创建规则失败: %s", err)
		ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: callback.GetQueryID(),
			Message: "创建规则失败",
			Alert:   true,
		})
		return dispatcher.EndGroups
	}

	// 清理向导状态和输入状态
	delete(ruleWizardStates, userChatID)
	delete(userInputStates, userChatID)

	// 显示成功消息
	template := msgelem.NewSuccessTemplate("✅ 规则创建成功", "新规则已添加到您的规则列表")
	
	template.AddItem(getRuleTypeIcon(newRule.Type), "规则类型", getRuleTypeName(newRule.Type), msgelem.ItemTypeText)
	template.AddItem("🔍", "匹配条件", newRule.Data, msgelem.ItemTypeCode)
	template.AddItem("📁", "存储位置", newRule.StorageName, msgelem.ItemTypeText)
	template.AddItem("📂", "目录路径", newRule.DirPath, msgelem.ItemTypeCode)

	keyboard := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "➕ 继续添加",
						Data: []byte("rule_add_start"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📋 查看列表",
						Data: []byte("rule_view_list"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回",
						Data: []byte("rule_back"),
					},
				},
			},
		},
	}

	messageText, entities := template.BuildFormattedMessage()

	ctx.EditMessage(callback.GetUserID(), &tg.MessagesEditMessageRequest{
		ID:        callback.MsgID,
		Message:   messageText,
		Entities:  entities,
		ReplyMarkup: keyboard,
	})

	ctx.AnswerCallback(&tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: callback.GetQueryID(),
		Message: "规则创建成功！",
	})

	return dispatcher.EndGroups
}

// getRuleTemplates 获取指定规则类型的模板
func getRuleTemplates(ruleType string) []ruletemplate.RuleTemplate {
	return ruletemplate.GetRuleTemplates(ruleType)
}