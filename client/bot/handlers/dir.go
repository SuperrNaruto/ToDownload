package handlers

import (
	"strconv"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/storage"
)

func handleDirCmd(ctx *ext.Context, update *ext.Update) error {
	logger := log.FromContext(ctx)
	args := strings.Split(update.EffectiveMessage.Text, " ")
	userChatID := update.GetUserChat().GetID()
	dirs, err := database.GetUserDirsByChatID(ctx, userChatID)
	if err != nil {
		logger.Errorf("获取用户文件夹失败: %s", err)
		errorTemplate := msgelem.NewErrorTemplate("获取文件夹失败", "无法加载用户的文件夹配置")

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if formatErr != nil {
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}
	if len(args) < 2 {
		ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
		return dispatcher.EndGroups
	}
	user, err := database.GetUserByChatID(ctx, update.GetUserChat().GetID())
	if err != nil {
		logger.Errorf("获取用户失败: %s", err)
		errorTemplate := msgelem.NewErrorTemplate("获取用户信息失败", "无法加载用户配置")

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if formatErr != nil {
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
		return dispatcher.EndGroups
	}
	switch args[1] {
	case "add":
		// /dir add local1 path/to/dir
		if len(args) < 4 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
			return dispatcher.EndGroups
		}
		if _, err := storage.Manager.GetUserStorageByName(ctx, user.ChatID, args[2]); err != nil {
			errorTemplate := msgelem.NewErrorTemplate("存储配置错误", err.Error())

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if formatErr != nil {
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}

		if err := database.CreateDirForUser(ctx, user.ID, args[2], args[3]); err != nil {
			logger.Errorf("创建文件夹失败: %s", err)
			errorTemplate := msgelem.NewErrorTemplate("创建文件夹失败", "无法添加新的文件夹配置")

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if formatErr != nil {
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}
		successTemplate := msgelem.NewSuccessTemplate("文件夹添加成功", "新的文件夹配置已经成功添加")

		// 使用格式化消息发送
		text, entities := successTemplate.BuildFormattedMessage()
		formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if formatErr != nil {
			ctx.Reply(update, ext.ReplyTextString(successTemplate.BuildMessage()), nil)
		}
	case "del":
		// /dir del 3
		if len(args) < 3 {
			ctx.Reply(update, ext.ReplyTextStyledTextArray(msgelem.BuildDirHelpStyling(dirs)), nil)
			return dispatcher.EndGroups
		}
		dirID, err := strconv.Atoi(args[2])
		if err != nil {
			errorTemplate := msgelem.NewErrorTemplate("无效参数", "文件夹ID必须是数字")

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if formatErr != nil {
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}
		if err := database.DeleteDirByID(ctx, uint(dirID)); err != nil {
			logger.Errorf("删除文件夹失败: %s", err)
			errorTemplate := msgelem.NewErrorTemplate("删除文件夹失败", "无法移除指定的文件夹配置")

			// 使用格式化消息发送
			text, entities := errorTemplate.BuildFormattedMessage()
			formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
			if formatErr != nil {
				ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
			}
			return dispatcher.EndGroups
		}
		successTemplate := msgelem.NewSuccessTemplate("文件夹删除成功", "指定的文件夹配置已被成功移除")

		// 使用格式化消息发送
		text, entities := successTemplate.BuildFormattedMessage()
		formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if formatErr != nil {
			ctx.Reply(update, ext.ReplyTextString(successTemplate.BuildMessage()), nil)
		}
	default:
		errorTemplate := msgelem.NewErrorTemplate("未知操作", "请使用 add 或 del 操作")

		// 使用格式化消息发送
		text, entities := errorTemplate.BuildFormattedMessage()
		formatErr := msgelem.ReplyWithFormattedText(ctx, update, text, entities, nil)
		if formatErr != nil {
			ctx.Reply(update, ext.ReplyTextString(errorTemplate.BuildMessage()), nil)
		}
	}
	return dispatcher.EndGroups
}
