package handlers

import (
	"fmt"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/krau/SaveAny-Bot/pkg/consts"
)

func handleHelpCmd(ctx *ext.Context, update *ext.Update) error {
	const helpText string = `
Save Any Bot - 转存你的 Telegram 文件
版本: %s , 提交: %s

基础命令:
/start - 开始使用
/help - 显示帮助
/silent - 开关静默模式
/storage - 设置默认存储位置
/save [自定义文件名] - 保存文件
/dir - 管理存储目录
/rule - 管理规则

AI智能重命名:
/ai_status - 查看AI功能状态
/ai_toggle - 开启/关闭AI重命名功能

使用帮助: https://sabot.unv.app/usage/
`
	shortHash := consts.GitCommit
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}
	ctx.Reply(update, ext.ReplyTextString(fmt.Sprintf(helpText, consts.Version, shortHash)), nil)
	return dispatcher.EndGroups
}
