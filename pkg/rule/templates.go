package rule

import (
	"fmt"

	"github.com/krau/SaveAny-Bot/pkg/enums/rule"
)

// RuleTemplate 规则模板结构
type RuleTemplate struct {
	ID          string
	Category    string
	Name        string
	Description string
	Icon        string
	Type        string
	DataPattern string
	Examples    []string
}

// GetRuleTemplates 获取指定类型的规则模板
func GetRuleTemplates(ruleType string) []RuleTemplate {
	switch ruleType {
	case rule.FileNameRegex.String():
		return getFilenameTemplates()
	case rule.MessageRegex.String():
		return getMessageTemplates()
	case rule.IsAlbum.String():
		return getAlbumTemplates()
	default:
		return []RuleTemplate{}
	}
}

// GetAllTemplates 获取所有规则模板
func GetAllTemplates() map[string][]RuleTemplate {
	return map[string][]RuleTemplate{
		rule.FileNameRegex.String(): getFilenameTemplates(),
		rule.MessageRegex.String():  getMessageTemplates(),
		rule.IsAlbum.String():       getAlbumTemplates(),
	}
}

// GetTemplateByID 根据ID获取模板
func GetTemplateByID(templateID string) *RuleTemplate {
	allTemplates := GetAllTemplates()
	for _, templates := range allTemplates {
		for _, template := range templates {
			if template.ID == templateID {
				return &template
			}
		}
	}
	return nil
}

// getFilenameTemplates 获取文件名匹配模板
func getFilenameTemplates() []RuleTemplate {
	return []RuleTemplate{
		{
			ID:          "filename_image",
			Category:    "媒体文件",
			Name:        "图片文件",
			Description: "匹配常见图片格式文件",
			Icon:        "🖼️",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(jpg|jpeg|png|gif|webp|bmp|svg)$`,
			Examples:    []string{"photo.jpg", "avatar.png", "image.gif"},
		},
		{
			ID:          "filename_video",
			Category:    "媒体文件",
			Name:        "视频文件",
			Description: "匹配常见视频格式文件",
			Icon:        "🎥",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(mp4|avi|mkv|mov|wmv|flv|webm|m4v)$`,
			Examples:    []string{"movie.mp4", "clip.avi", "video.mkv"},
		},
		{
			ID:          "filename_audio",
			Category:    "媒体文件",
			Name:        "音频文件",
			Description: "匹配常见音频格式文件",
			Icon:        "🎵",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(mp3|flac|wav|aac|ogg|wma|m4a)$`,
			Examples:    []string{"song.mp3", "music.flac", "audio.wav"},
		},
		{
			ID:          "filename_document",
			Category:    "文档文件",
			Name:        "文档文件",
			Description: "匹配常见文档格式文件",
			Icon:        "📄",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(pdf|doc|docx|txt|rtf|odt)$`,
			Examples:    []string{"document.pdf", "report.docx", "notes.txt"},
		},
		{
			ID:          "filename_archive",
			Category:    "压缩文件",
			Name:        "压缩文件",
			Description: "匹配常见压缩格式文件",
			Icon:        "📦",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(zip|rar|7z|tar|gz|bz2|xz)$`,
			Examples:    []string{"archive.zip", "backup.rar", "files.7z"},
		},
		{
			ID:          "filename_code",
			Category:    "代码文件",
			Name:        "代码文件",
			Description: "匹配常见编程语言文件",
			Icon:        "💻",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(js|ts|py|go|java|cpp|c|h|php|rb|rs|swift)$`,
			Examples:    []string{"script.js", "main.py", "app.go"},
		},
		{
			ID:          "filename_ebook",
			Category:    "电子书",
			Name:        "电子书文件",
			Description: "匹配常见电子书格式文件",
			Icon:        "📚",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(epub|mobi|azw|azw3|fb2|chm)$`,
			Examples:    []string{"book.epub", "novel.mobi", "guide.azw3"},
		},
		{
			ID:          "filename_spreadsheet",
			Category:    "表格文件",
			Name:        "表格文件",
			Description: "匹配常见表格格式文件",
			Icon:        "📊",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(xls|xlsx|csv|ods)$`,
			Examples:    []string{"data.xlsx", "report.csv", "table.ods"},
		},
		{
			ID:          "filename_presentation",
			Category:    "演示文件",
			Name:        "演示文稿",
			Description: "匹配常见演示格式文件",
			Icon:        "📺",
			Type:        rule.FileNameRegex.String(),
			DataPattern: `\.(ppt|pptx|odp|key)$`,
			Examples:    []string{"slides.pptx", "presentation.ppt", "keynote.key"},
		},
	}
}

// getMessageTemplates 获取消息内容匹配模板
func getMessageTemplates() []RuleTemplate {
	return []RuleTemplate{
		{
			ID:          "message_link",
			Category:    "链接内容",
			Name:        "链接消息",
			Description: "匹配包含HTTP/HTTPS链接的消息",
			Icon:        "🔗",
			Type:        rule.MessageRegex.String(),
			DataPattern: `https?://[^\s]+`,
			Examples:    []string{"查看这个网站 https://example.com", "https://github.com/user/repo"},
		},
		{
			ID:          "message_email",
			Category:    "联系信息",
			Name:        "邮箱地址",
			Description: "匹配包含邮箱地址的消息",
			Icon:        "📧",
			Type:        rule.MessageRegex.String(),
			DataPattern: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			Examples:    []string{"联系邮箱：user@example.com", "发送到 admin@domain.org"},
		},
		{
			ID:          "message_phone",
			Category:    "联系信息",
			Name:        "电话号码",
			Description: "匹配包含电话号码的消息",
			Icon:        "📞",
			Type:        rule.MessageRegex.String(),
			DataPattern: `(?:(?:\+86)|(?:86))?\s*1[3-9]\d{9}`,
			Examples:    []string{"联系电话：13800138000", "+86 15912345678"},
		},
		{
			ID:          "message_keyword_important",
			Category:    "关键词匹配",
			Name:        "重要消息",
			Description: "匹配包含重要关键词的消息",
			Icon:        "⚡",
			Type:        rule.MessageRegex.String(),
			DataPattern: `(?i)(重要|紧急|urgent|important|critical|高优先级)`,
			Examples:    []string{"重要通知", "紧急文件", "Important document"},
		},
		{
			ID:          "message_keyword_work",
			Category:    "关键词匹配",
			Name:        "工作相关",
			Description: "匹配包含工作关键词的消息",
			Icon:        "💼",
			Type:        rule.MessageRegex.String(),
			DataPattern: `(?i)(工作|项目|会议|报告|文档|work|project|meeting|report|document)`,
			Examples:    []string{"工作文档", "项目报告", "Meeting notes"},
		},
		{
			ID:          "message_keyword_study",
			Category:    "关键词匹配",
			Name:        "学习资料",
			Description: "匹配包含学习关键词的消息",
			Icon:        "📚",
			Type:        rule.MessageRegex.String(),
			DataPattern: `(?i)(学习|教程|课程|资料|笔记|study|tutorial|course|notes|learning)`,
			Examples:    []string{"学习资料", "教程视频", "Course material"},
		},
		{
			ID:          "message_date",
			Category:    "时间日期",
			Name:        "包含日期",
			Description: "匹配包含日期格式的消息",
			Icon:        "📅",
			Type:        rule.MessageRegex.String(),
			DataPattern: `\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日]?`,
			Examples:    []string{"2024-01-15 的报告", "2024年1月15日会议", "2024/01/15"},
		},
	}
}

// getAlbumTemplates 获取相册文件匹配模板
func getAlbumTemplates() []RuleTemplate {
	return []RuleTemplate{
		{
			ID:          "album_photos",
			Category:    "相册处理",
			Name:        "相册图片",
			Description: "自动匹配和整理相册中的图片文件",
			Icon:        "📸",
			Type:        rule.IsAlbum.String(),
			DataPattern: "true", // IS-ALBUM 类型使用固定值
			Examples:    []string{"多张图片作为相册发送", "批量图片上传"},
		},
	}
}

// GetTemplateCategoryIcon 获取分类图标
func GetTemplateCategoryIcon(category string) string {
	icons := map[string]string{
		"媒体文件": "🎬",
		"文档文件": "📄",
		"压缩文件": "📦",
		"代码文件": "💻",
		"电子书":  "📚",
		"表格文件": "📊",
		"演示文件": "📺",
		"链接内容": "🔗",
		"联系信息": "📞",
		"关键词匹配": "🔍",
		"时间日期": "📅",
		"相册处理": "📸",
	}
	
	if icon, exists := icons[category]; exists {
		return icon
	}
	return "📁"
}

// ValidateTemplate 验证模板数据
func ValidateTemplate(template *RuleTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("模板ID不能为空")
	}
	if template.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	if template.Type == "" {
		return fmt.Errorf("模板类型不能为空")
	}
	if template.DataPattern == "" {
		return fmt.Errorf("模板数据模式不能为空")
	}
	return nil
}

// GetTemplatesByCategory 按分类获取模板
func GetTemplatesByCategory() map[string][]RuleTemplate {
	allTemplates := GetAllTemplates()
	categoryMap := make(map[string][]RuleTemplate)
	
	for _, templates := range allTemplates {
		for _, template := range templates {
			categoryMap[template.Category] = append(categoryMap[template.Category], template)
		}
	}
	
	return categoryMap
}