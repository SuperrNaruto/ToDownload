package msgelem

import (
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

// BuildFormattedMessage 构建格式化消息（支持样式组件）
func BuildFormattedMessage(parts ...styling.StyledTextOption) (string, []tg.MessageEntityClass) {
	entityBuilder := entity.Builder{}

	err := styling.Perform(&entityBuilder, parts...)
	if err != nil {
		// 如果格式化失败，尝试提取纯文本
		plainText := ""
		for _, part := range parts {
			plainText += extractPlainText(part)
		}
		// 如果还是没有文本，提供一个基本的回退消息
		if plainText == "" {
			plainText = "[格式化失败，请稍后重试]"
		}
		return plainText, nil
	}

	formattedText, entities := entityBuilder.Complete()
	return formattedText, entities
}

// extractPlainText 从styling选项中提取纯文本（fallback用）
func extractPlainText(option styling.StyledTextOption) string {
	// 创建一个临时的entity builder来提取纯文本
	tempBuilder := entity.Builder{}
	err := styling.Perform(&tempBuilder, option)
	if err != nil {
		return ""
	}
	text, _ := tempBuilder.Complete()
	return text
}

// BuildHelpMessage 构建格式化的帮助消息
func BuildHelpMessage(title, subtitle string, sections []HelpSection) (string, []tg.MessageEntityClass) {
	var parts []styling.StyledTextOption

	// 添加标题
	parts = append(parts,
		styling.Bold("🤖 "+title),
		styling.Plain("\n"+subtitle+"\n\n"),
	)

	// 添加各个章节
	for i, section := range sections {
		if i > 0 {
			parts = append(parts, styling.Plain("\n"))
		}

		// 章节标题
		parts = append(parts,
			styling.Bold(section.Icon+" "+section.Title),
			styling.Plain("\n\n"),
		)

		// 章节内容
		for _, item := range section.Items {
			parts = append(parts,
				styling.Plain("• "+item+"\n"),
			)
		}
	}

	return BuildFormattedMessage(parts...)
}

// HelpSection 帮助章节结构
type HelpSection struct {
	Icon  string
	Title string
	Items []string
}

// BuildFormattedConfigMessage 构建格式化的配置消息
func BuildFormattedConfigMessage(title string, configs map[string]string) (string, []tg.MessageEntityClass) {
	var parts []styling.StyledTextOption

	parts = append(parts,
		styling.Bold("⚙️ "+title),
		styling.Plain("\n\n"),
	)

	for key, value := range configs {
		parts = append(parts,
			styling.Plain("• "),
			styling.Bold(key),
			styling.Plain(": "),
			styling.Code(value),
			styling.Plain("\n"),
		)
	}

	return BuildFormattedMessage(parts...)
}

// BuildStatusMessage 构建格式化的状态消息
func BuildStatusMessage(title string, items []StatusItem) (string, []tg.MessageEntityClass) {
	var parts []styling.StyledTextOption

	parts = append(parts,
		styling.Bold("📊 "+title),
		styling.Plain("\n\n"),
	)

	for _, item := range items {
		statusIcon := "✅"
		if !item.Success {
			statusIcon = "❌"
		}

		parts = append(parts,
			styling.Plain(statusIcon+" "),
			styling.Bold(item.Name),
			styling.Plain(": "+item.Value+"\n"),
		)
	}

	return BuildFormattedMessage(parts...)
}

// StatusItem 状态项结构
type StatusItem struct {
	Name    string
	Value   string
	Success bool
}

// FormatProgressBarFormatted 创建带格式化的进度条（用于BuildFormattedMessage）
func FormatProgressBarFormatted(processedBytes, totalBytes int64, barLength int) []styling.StyledTextOption {
	if totalBytes <= 0 {
		emptyBar := strings.Repeat("░", barLength)
		return []styling.StyledTextOption{
			styling.Plain(emptyBar + " "),
			styling.Bold("0.0%"),
		}
	}

	percent := float64(processedBytes) / float64(totalBytes) * 100

	// 计算填充的字符数
	filled := int(percent * float64(barLength) / 100)
	if filled > barLength {
		filled = barLength
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

	return []styling.StyledTextOption{
		styling.Plain(bar + " "),
		styling.Bold(fmt.Sprintf("%.1f%%", percent)),
	}
}
