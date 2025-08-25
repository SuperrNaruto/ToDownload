package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/rule"
)

func handleRuleCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	userChatID := update.GetUserChat().GetID()
	user, err := database.GetUserByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("获取用户规则失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("获取用户规则失败"), nil)
		return dispatcher.EndGroups
	}

	// 如果没有参数或只有 /rule，显示新的图形界面
	if len(args) < 2 {
		return showRuleManagementInterface(ctx, update, user)
	}

	// 保持向后兼容的命令行界面
	switch args[1] {
	case "switch":
		// /rule switch
		applyRule := !user.ApplyRule
		if err := database.UpdateUserApplyRule(ctx, user.ChatID, applyRule); err != nil {
			logger.Errorf("更新用户失败: %s", err)
			ctx.Reply(update, ext.ReplyTextString("更新用户失败"), nil)
			return dispatcher.EndGroups
		}
		ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf("已%s规则模式", map[bool]string{true: "启用", false: "禁用"}[applyRule])), nil)
	case "add":
		// /rule add <type> <data> <storage> <dirpath>
		if len(args) < 6 {
			return showRuleManagementInterface(ctx, update, user)
		}
		return handleLegacyRuleAdd(ctx, update, user, args)
	case "del":
		// /rule del <id>
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextString("请提供规则ID"), nil)
			return dispatcher.EndGroups
		}
		return handleLegacyRuleDelete(ctx, update, args)
	default:
		return showRuleManagementInterface(ctx, update, user)
	}
	return dispatcher.EndGroups
}

// showRuleManagementInterface 显示新的规则管理界面
func showRuleManagementInterface(ctx *ext.Context, update *ext.Update, user *database.User) error {
	template := msgelem.NewInfoTemplate("📋 规则管理", "管理您的自动文件组织规则")
	
	// 显示规则模式状态
	ruleStatusText := "已禁用"
	if user.ApplyRule {
		ruleStatusText = "已启用"
	}
	template.AddItem("⚙️", "规则模式", ruleStatusText, msgelem.ItemTypeStatus)
	template.AddItem("📊", "规则数量", fmt.Sprintf("%d", len(user.Rules)), msgelem.ItemTypeText)

	// 如果有规则，显示最近添加的规则
	if len(user.Rules) > 0 {
		lastRule := user.Rules[len(user.Rules)-1]
		template.AddItem("📝", "最新规则", fmt.Sprintf("%s -> %s", lastRule.Type, lastRule.StorageName), msgelem.ItemTypeText)
	}

	template.AddAction("使用下方按钮管理您的规则")

	// 构建内联键盘
	keyboard := buildRuleManagementKeyboard(user.ApplyRule, len(user.Rules))

	messageText, entities := template.BuildFormattedMessage()

	err := msgelem.ReplyWithFormattedText(ctx, update, messageText, entities, &ext.ReplyOpts{
		Markup: keyboard,
	})
	if err != nil {
		// 如果格式化发送失败，回退到普通发送
		ctx.Reply(update, ext.ReplyTextString(template.BuildMessage()), &ext.ReplyOpts{
			Markup: keyboard,
		})
	}

	return dispatcher.EndGroups
}

// handleLegacyRuleAdd 处理旧版命令行添加规则
func handleLegacyRuleAdd(ctx *ext.Context, update *ext.Update, user *database.User, args []string) error {
	logger := log.FromContext(ctx)
	
	ruleTypeArg := args[2]
	ruleType, err := func() (rule.RuleType, error) {
		for _, t := range rule.Values() {
			if strings.EqualFold(t.String(), ruleTypeArg) {
				return t, nil
			}
		}
		return rule.RuleType(""), fmt.Errorf("无效的规则类型: %s\n可用: %v", ruleTypeArg, slice.Join(rule.Values(), ", "))
	}()
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString(err.Error()), nil)
		return dispatcher.EndGroups
	}

	ruleData := args[3]
	storageName := args[4]
	dirPath := args[5]

	rd := &database.Rule{
		Type:        ruleType.String(),
		Data:        ruleData,
		StorageName: storageName,
		DirPath:     dirPath,
		UserID:      user.ID,
	}
	if err := database.CreateRule(ctx, rd); err != nil {
		logger.Errorf("创建规则失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("创建规则失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("创建规则成功"), nil)
	return dispatcher.EndGroups
}

// handleLegacyRuleDelete 处理旧版命令行删除规则
func handleLegacyRuleDelete(ctx *ext.Context, update *ext.Update, args []string) error {
	logger := log.FromContext(ctx)
	
	ruleID := args[2]
	id, err := strconv.Atoi(ruleID)
	if err != nil {
		ctx.Reply(update, ext.ReplyTextString("无效的规则ID"), nil)
		return dispatcher.EndGroups
	}
	if err := database.DeleteRule(ctx, uint(id)); err != nil {
		logger.Errorf("删除规则失败: %s", err)
		ctx.Reply(update, ext.ReplyTextString("删除规则失败"), nil)
		return dispatcher.EndGroups
	}
	ctx.Reply(update, ext.ReplyTextString("删除规则成功"), nil)
	return dispatcher.EndGroups
}
