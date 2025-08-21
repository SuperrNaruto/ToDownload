package msgelem

import (
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/tg"
)

// ReplyWithFormattedText 发送格式化的回复消息
func ReplyWithFormattedText(ctx *ext.Context, update *ext.Update, text string, entities []tg.MessageEntityClass, opts *ext.ReplyOpts) error {
	chatID := update.GetUserChat().GetID()
	
	// 构建发送请求
	req := &tg.MessagesSendMessageRequest{
		Message: text,
	}
	
	// 设置格式化实体（这是关键！）
	if len(entities) > 0 {
		req.SetEntities(entities)
	}
	
	// 设置回复标记
	if opts != nil && opts.Markup != nil {
		req.SetReplyMarkup(opts.Markup)
	}
	
	// 发送带格式化的消息
	_, err := ctx.SendMessage(chatID, req)
	if err != nil {
		// 如果格式化发送失败，回退到普通发送
		_, fallbackErr := ctx.Reply(update, ext.ReplyTextString(text), opts)
		return fallbackErr
	}
	
	return nil
}

// EditWithFormattedText 编辑消息为格式化文本
func EditWithFormattedText(ctx *ext.Context, peer tg.InputPeerClass, msgID int, text string, entities []tg.MessageEntityClass, markup tg.ReplyMarkupClass) error {
	// 使用正确的方式设置entities
	userPeer := peer.(*tg.InputPeerUser)
	req := &tg.MessagesEditMessageRequest{
		ID: msgID,
	}
	
	// 使用setter方法来设置字段，这样可以确保正确处理
	req.SetMessage(text)
	req.SetEntities(entities)
	if markup != nil {
		req.SetReplyMarkup(markup)
	}
	
	_, err := ctx.EditMessage(userPeer.UserID, req)
	return err
}

// SendFormattedMessage 发送格式化消息到指定用户
func SendFormattedMessage(ctx *ext.Context, userID int64, text string, entities []tg.MessageEntityClass, markup tg.ReplyMarkupClass) error {
	req := &tg.MessagesSendMessageRequest{
		Message: text,
	}
	
	// 设置格式化实体
	if len(entities) > 0 {
		req.SetEntities(entities)
	}
	
	// 设置回复标记
	if markup != nil {
		req.SetReplyMarkup(markup)
	}
	
	_, err := ctx.SendMessage(userID, req)
	return err
}