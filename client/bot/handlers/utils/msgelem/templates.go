package msgelem

import (
	"fmt"
	"strings"
	"time"

	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
)

// MessageTemplate 统一消息模板系统
type MessageTemplate struct {
	Title       string
	Description string
	Status      string
	Items       []TemplateItem
	Actions     []string
	Footer      string
}

// TemplateItem 消息项目
type TemplateItem struct {
	Icon  string
	Label string
	Value string
	Type  ItemType
}

// ItemType 项目类型
type ItemType string

const (
	ItemTypeText     ItemType = "text"
	ItemTypeCode     ItemType = "code"
	ItemTypeTime     ItemType = "time"
	ItemTypeSize     ItemType = "size"
	ItemTypeStatus   ItemType = "status"
	ItemTypeProgress ItemType = "progress"
)

// StatusIcon 状态图标
func StatusIcon(status string) string {
	switch status {
	case "success":
		return "✅"
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	case "processing":
		return "⏳"
	case "pending":
		return "🕐"
	default:
		return "•"
	}
}

// FormatSize 格式化文件大小
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 格式化时间间隔
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// BuildMessage 构建格式化消息
func (t *MessageTemplate) BuildMessage() string {
	var msg strings.Builder

	// 标题部分
	if t.Title != "" {
		if t.Status != "" {
			msg.WriteString(fmt.Sprintf("%s **%s**\n", StatusIcon(t.Status), t.Title))
		} else {
			msg.WriteString(fmt.Sprintf("**%s**\n", t.Title))
		}
	}

	// 描述部分
	if t.Description != "" {
		msg.WriteString(fmt.Sprintf("%s\n", t.Description))
	}

	// 分隔线
	if len(t.Items) > 0 {
		msg.WriteString("\n")
	}

	// 项目列表
	for _, item := range t.Items {
		line := t.formatItem(item)
		if line != "" {
			msg.WriteString(line + "\n")
		}
	}

	// 操作说明
	if len(t.Actions) > 0 {
		msg.WriteString("\n")
		for _, action := range t.Actions {
			msg.WriteString(fmt.Sprintf("💡 %s\n", action))
		}
	}

	// 页脚
	if t.Footer != "" {
		msg.WriteString(fmt.Sprintf("\n%s", t.Footer))
	}

	return msg.String()
}

// formatItem 格式化单个项目
func (t *MessageTemplate) formatItem(item TemplateItem) string {
	if item.Value == "" {
		return ""
	}

	switch item.Type {
	case ItemTypeCode:
		return fmt.Sprintf("%s **%s**: `%s`", item.Icon, item.Label, item.Value)
	case ItemTypeTime:
		if parsedTime, err := time.Parse(time.RFC3339, item.Value); err == nil {
			return fmt.Sprintf("%s **%s**: %s", item.Icon, item.Label, parsedTime.Format("2006-01-02 15:04"))
		}
		return fmt.Sprintf("%s **%s**: %s", item.Icon, item.Label, item.Value)
	case ItemTypeSize:
		if size, ok := parseSize(item.Value); ok {
			return fmt.Sprintf("%s **%s**: %s", item.Icon, item.Label, FormatSize(size))
		}
		return fmt.Sprintf("%s **%s**: %s", item.Icon, item.Label, item.Value)
	case ItemTypeStatus:
		statusIcon := StatusIcon(item.Value)
		return fmt.Sprintf("%s **%s**: %s %s", item.Icon, item.Label, statusIcon, item.Value)
	case ItemTypeProgress:
		return fmt.Sprintf("%s **%s**: %s", item.Icon, item.Label, formatProgress(item.Value))
	default:
		return fmt.Sprintf("%s **%s**: %s", item.Icon, item.Label, item.Value)
	}
}

// parseSize 解析大小字符串
func parseSize(value string) (int64, bool) {
	var size int64
	_, err := fmt.Sscanf(value, "%d", &size)
	return size, err == nil
}

// formatProgress 格式化进度条
func formatProgress(value string) string {
	var current, total int
	if n, err := fmt.Sscanf(value, "%d/%d", &current, &total); n == 2 && err == nil {
		return FormatProgressBarWithLength(int64(current), int64(total), 20)
	}
	return value
}

// FormatProgressBarWithLength 创建指定长度的进度条
func FormatProgressBarWithLength(processedBytes, totalBytes int64, barLength int) string {
	if totalBytes <= 0 {
		emptyBar := strings.Repeat("░", barLength)
		return fmt.Sprintf("%s **0.0%%**", emptyBar)
	}

	percent := float64(processedBytes) / float64(totalBytes) * 100

	// 计算填充的字符数
	filled := int(percent * float64(barLength) / 100)
	if filled > barLength {
		filled = barLength
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

	return fmt.Sprintf("%s **%.1f%%**", bar, percent)
}

// 预定义模板构建器

// NewSuccessTemplate 成功消息模板
func NewSuccessTemplate(title, description string) *MessageTemplate {
	return &MessageTemplate{
		Title:       title,
		Description: description,
		Status:      "success",
	}
}

// NewErrorTemplate 错误消息模板
func NewErrorTemplate(title, description string) *MessageTemplate {
	return &MessageTemplate{
		Title:       title,
		Description: description,
		Status:      "error",
	}
}

// NewInfoTemplate 信息消息模板
func NewInfoTemplate(title, description string) *MessageTemplate {
	return &MessageTemplate{
		Title:       title,
		Description: description,
		Status:      "info",
	}
}

// NewProcessingTemplate 处理中消息模板
func NewProcessingTemplate(title, description string) *MessageTemplate {
	return &MessageTemplate{
		Title:       title,
		Description: description,
		Status:      "processing",
	}
}

// AddProgressBar 添加格式化的进度条
func (t *MessageTemplate) AddProgressBar(icon, label string, current, total int64, barLength int) *MessageTemplate {
	t.Items = append(t.Items, TemplateItem{
		Icon:  icon,
		Label: label,
		Value: fmt.Sprintf("%d|%d|%d", current, total, barLength), // 特殊格式用于识别
		Type:  ItemTypeProgress,
	})
	return t
}

// AddItem 添加项目
func (t *MessageTemplate) AddItem(icon, label, value string, itemType ItemType) *MessageTemplate {
	t.Items = append(t.Items, TemplateItem{
		Icon:  icon,
		Label: label,
		Value: value,
		Type:  itemType,
	})
	return t
}

// AddAction 添加操作说明
func (t *MessageTemplate) AddAction(action string) *MessageTemplate {
	t.Actions = append(t.Actions, action)
	return t
}

// SetFooter 设置页脚
func (t *MessageTemplate) SetFooter(footer string) *MessageTemplate {
	t.Footer = footer
	return t
}

// 常用消息构建器

// BuildStorageStatusMessage 构建存储状态消息
func BuildStorageStatusMessage(storageName, storageType, status string, isDefault bool) string {
	template := NewInfoTemplate("存储状态", "")

	template.AddItem("📁", "存储名称", storageName, ItemTypeText)
	template.AddItem("🔧", "存储类型", storageType, ItemTypeText)
	template.AddItem("📊", "状态", status, ItemTypeStatus)

	if isDefault {
		template.AddItem("⭐", "默认存储", "是", ItemTypeText)
	}

	return template.BuildMessage()
}

// BuildFileInfoMessage 构建文件信息消息
func BuildFileInfoMessage(fileName string, fileSize int64, mimeType string) string {
	template := NewInfoTemplate("文件信息", "")

	template.AddItem("📄", "文件名", fileName, ItemTypeCode)
	template.AddItem("📏", "文件大小", fmt.Sprintf("%d", fileSize), ItemTypeSize)
	template.AddItem("🏷️", "文件类型", mimeType, ItemTypeText)

	return template.BuildMessage()
}

// BuildTaskProgressMessage 构建任务进度消息
func BuildTaskProgressMessage(taskName string, current, total int, status string) string {
	template := NewProcessingTemplate("任务进度", taskName)

	template.AddItem("📊", "进度", fmt.Sprintf("%d/%d", current, total), ItemTypeProgress)
	template.AddItem("📱", "状态", status, ItemTypeStatus)

	return template.BuildMessage()
}

// BuildConfigMessage 构建配置消息
func BuildConfigMessage(title string, configs map[string]string) string {
	template := NewInfoTemplate(title, "当前配置详情：")

	for key, value := range configs {
		icon := "⚙️"
		switch key {
		case "url", "endpoint":
			icon = "🌐"
		case "username", "user":
			icon = "👤"
		case "path", "directory":
			icon = "📁"
		case "enable", "enabled":
			icon = "🔘"
		}
		template.AddItem(icon, key, value, ItemTypeText)
	}

	return template.BuildMessage()
}

// BuildFormattedMessage 构建格式化消息（使用 Telegram 实体）
func (t *MessageTemplate) BuildFormattedMessage() (string, []tg.MessageEntityClass) {
	var parts []styling.StyledTextOption

	// 标题部分
	if t.Title != "" {
		if t.Status != "" {
			parts = append(parts,
				styling.Plain(StatusIcon(t.Status)+" "),
				styling.Bold(t.Title),
				styling.Plain("\n"),
			)
		} else {
			parts = append(parts,
				styling.Bold(t.Title),
				styling.Plain("\n"),
			)
		}
	}

	// 描述部分
	if t.Description != "" {
		parts = append(parts,
			styling.Plain(t.Description+"\n"),
		)
	}

	// 分隔线
	if len(t.Items) > 0 {
		parts = append(parts, styling.Plain("\n"))
	}

	// 项目列表
	for _, item := range t.Items {
		itemParts := t.formatItemFormatted(item)
		if len(itemParts) > 0 {
			parts = append(parts, itemParts...)
			parts = append(parts, styling.Plain("\n"))
		}
	}

	// 操作说明
	if len(t.Actions) > 0 {
		parts = append(parts, styling.Plain("\n"))
		for _, action := range t.Actions {
			parts = append(parts,
				styling.Plain("💡 "+action+"\n"),
			)
		}
	}

	// 页脚
	if t.Footer != "" {
		parts = append(parts,
			styling.Plain("\n"+t.Footer),
		)
	}

	return BuildFormattedMessage(parts...)
}

// formatItemFormatted 格式化单个项目（用于格式化版本）
func (t *MessageTemplate) formatItemFormatted(item TemplateItem) []styling.StyledTextOption {
	if item.Value == "" {
		return nil
	}

	var parts []styling.StyledTextOption

	// 计算最大标签长度进行对齐
	maxLabelLength := t.getMaxLabelLength()
	paddedLabel := t.padLabel(item.Label, maxLabelLength)

	// 图标和标签
	parts = append(parts,
		styling.Plain(item.Icon+" "),
		styling.Bold(paddedLabel+": "),
	)

	// 根据类型格式化值
	switch item.Type {
	case ItemTypeCode:
		parts = append(parts, styling.Code(item.Value))
	case ItemTypeTime:
		if parsedTime, err := time.Parse(time.RFC3339, item.Value); err == nil {
			parts = append(parts, styling.Plain(parsedTime.Format("2006-01-02 15:04")))
		} else {
			parts = append(parts, styling.Plain(item.Value))
		}
	case ItemTypeSize:
		if size, ok := parseSize(item.Value); ok {
			parts = append(parts, styling.Plain(FormatSize(size)))
		} else {
			parts = append(parts, styling.Plain(item.Value))
		}
	case ItemTypeStatus:
		statusIcon := StatusIcon(item.Value)
		parts = append(parts, styling.Plain(statusIcon+" "+item.Value))
	case ItemTypeProgress:
		// 解析进度条数据
		var current, total, barLength int64
		if n, err := fmt.Sscanf(item.Value, "%d|%d|%d", &current, &total, &barLength); n == 3 && err == nil {
			// 使用格式化的进度条
			progressParts := FormatProgressBarFormatted(current, total, int(barLength))
			parts = append(parts, progressParts...)
		} else {
			// 回退到普通文本
			parts = append(parts, styling.Plain(formatProgress(item.Value)))
		}
	default:
		parts = append(parts, styling.Plain(item.Value))
	}

	return parts
}

// getMaxLabelLength 计算所有项目中标签的最大长度
func (t *MessageTemplate) getMaxLabelLength() int {
	maxLength := 0
	for _, item := range t.Items {
		// 计算中文字符长度（中文字符占用更多显示宽度）
		length := calculateDisplayWidth(item.Label)
		if length > maxLength {
			maxLength = length
		}
	}
	return maxLength
}

// padLabel 填充标签以对齐
func (t *MessageTemplate) padLabel(label string, maxLength int) string {
	currentLength := calculateDisplayWidth(label)
	if currentLength >= maxLength {
		return label
	}
	
	// 使用空格填充到指定长度
	padding := maxLength - currentLength
	return label + strings.Repeat(" ", padding)
}

// calculateDisplayWidth 计算字符串的显示宽度（针对Telegram优化）
func calculateDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		// 针对Telegram客户端调整字符宽度计算
		if r > 127 {
			// 中文字符和全角字符在Telegram中占用2个显示宽度
			width += 2
		} else {
			width += 1
		}
	}
	// 为Telegram客户端添加额外的4个空格填充（原来是2个，现在增加到4个）
	return width + 4
}
