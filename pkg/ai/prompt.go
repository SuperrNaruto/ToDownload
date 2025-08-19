package ai

import (
	"fmt"
	"strings"
)

const (
	// BaseRenamePrompt is the base prompt template for file renaming
	BaseRenamePrompt = `你是一个专业的文件重命名助手。请根据提供的文件信息，生成一个规范化的文件名。

重命名规则：
1. 格式：名称.作者.时间.要点（各部分如果存在才包含）
2. 分隔符使用规则：
   - 各主要部分（名称、作者、时间、要点）之间必须用英文句点(.)分隔
   - 每个部分内部的多个词用下划线(_)连接
3. 各部分说明：
   - 名称：文档/视频/图片的主要标题或主题
   - 作者：创作者、发布者或机构名称  
   - 时间：相关的时间信息（日期、年份等）
   - 要点：关键信息、版本号、类型等重要标识

格式示例：
✅ 正确：游戏截图.任天堂.2023.塞尔达传说
✅ 正确：产品发布会.苹果公司.2024.iPhone_16
❌ 错误：游戏截图_任天堂.2023.塞尔达传说 (混合使用分隔符)
❌ 错误：游戏截图.任天堂_2023.塞尔达传说 (分隔符使用错误)

限制条件：
- 总长度不超过100个字符
- 避免使用特殊字符：/ \ : * ? " < > |
- 严格按照分隔符规则，不可混用
- 如果某部分信息不存在，直接省略（不要用"未知"等占位符）
- 保持简洁，优先保留最重要的信息

请仅返回重命名后的文件名（不包含文件扩展名），不要添加任何解释。

文件信息：
原文件名：%s
消息内容：%s`

	// AlbumRenamePrompt is the prompt template for album file renaming
	AlbumRenamePrompt = `你是一个专业的文件重命名助手。请为相册（媒体组）生成一个统一的基础文件名。

重命名规则：
1. 格式：名称.作者.时间.要点（各部分如果存在才包含）
2. 分隔符使用规则：
   - 各主要部分（名称、作者、时间、要点）之间必须用英文句点(.)分隔
   - 每个部分内部的多个词用下划线(_)连接
3. 这个名称将作为相册中所有文件的基础名称，后面会加上序号（如 _01, _02）
4. 各部分说明：
   - 名称：相册的主要标题或主题
   - 作者：创作者、发布者或机构名称  
   - 时间：相关的时间信息（日期、年份等）
   - 要点：关键信息、版本号、类型等重要标识

格式示例：
✅ 正确：风景照片.颐和园.春天.2024
✅ 正确：付费卡.NIKKE.2023.Alice
❌ 错误：风景照片_颐和园.春天.2024 (混合使用分隔符)
❌ 错误：付费卡.NIKKE_2023.Alice (分隔符使用错误)

限制条件：
- 总长度不超过80个字符（为序号预留空间）
- 避免使用特殊字符：/ \ : * ? " < > |
- 严格按照分隔符规则，不可混用
- 如果某部分信息不存在，直接省略
- 重点关注相册整体的主题，而非单个文件
- 保持简洁，优先保留最重要的信息

请仅返回重命名后的基础文件名（不包含文件扩展名和序号），不要添加任何解释。

相册信息：
消息内容：%s`
)

// RenameRequest represents a file rename request
type RenameRequest struct {
	OriginalFilename string
	MessageContent   string
	IsAlbum          bool
}

// BuildPrompt builds the appropriate prompt based on the request type
func BuildPrompt(req RenameRequest) string {
	// Clean the inputs
	originalFilename := strings.TrimSpace(req.OriginalFilename)
	messageContent := strings.TrimSpace(req.MessageContent)

	// Limit message content length to avoid token overflow
	if len(messageContent) > 1000 {
		messageContent = messageContent[:1000] + "..."
	}

	if req.IsAlbum {
		return fmt.Sprintf(AlbumRenamePrompt, messageContent)
	}

	return fmt.Sprintf(BaseRenamePrompt, originalFilename, messageContent)
}

// ValidateFilename checks if the generated filename is valid
func ValidateFilename(filename string) bool {
	if len(filename) == 0 || len(filename) > 100 {
		return false
	}

	// Check for invalid characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	return true
}

// SanitizeFilename removes or replaces invalid characters in filename
func SanitizeFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	
	// Replace invalid characters with underscores
	replacements := map[string]string{
		"/":  "_",
		"\\": "_",
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
		// 额外的路径安全字符处理
		"..": "_",    // 防止目录遍历
		"#":  "_",    // URL片段标识符
		"%":  "_",    // URL编码字符
		"&":  "_",    // URL参数分隔符
		"+":  "_",    // URL空格编码
		"=":  "_",    // URL参数赋值符
	}

	for old, new := range replacements {
		filename = strings.ReplaceAll(filename, old, new)
	}

	// Remove multiple consecutive underscores
	for strings.Contains(filename, "__") {
		filename = strings.ReplaceAll(filename, "__", "_")
	}

	// Trim underscores from start and end
	filename = strings.Trim(filename, "_")
	
	// 防止空文件名
	if filename == "" {
		filename = "untitled"
	}

	// Ensure length limit (为了兼容不同文件系统，限制为200字符)
	if len(filename) > 200 {
		// 尝试保留文件扩展名
		if lastDot := strings.LastIndex(filename, "."); lastDot > 0 && lastDot < 200 {
			ext := filename[lastDot:]
			if len(ext) < 10 { // 合理的扩展名长度
				maxBase := 200 - len(ext)
				filename = filename[:maxBase] + ext
			} else {
				filename = filename[:200]
			}
		} else {
			filename = filename[:200]
		}
	}

	return filename
}

// ValidateStoragePath 验证完整存储路径的安全性
func ValidateStoragePath(storagePath string) error {
	// 检查路径长度
	if len(storagePath) > 1000 {
		return fmt.Errorf("storage path too long: %d characters", len(storagePath))
	}
	
	// 检查危险的路径模式
	dangerousPatterns := []string{
		"../",         // 目录遍历
		"..\\",        // Windows目录遍历
		"//",          // 双斜杠可能导致URL解析问题
		"\\\\",        // 双反斜杠
		"./",          // 当前目录引用（在某些上下文中可能有问题）
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(storagePath, pattern) {
			return fmt.Errorf("storage path contains dangerous pattern: %s", pattern)
		}
	}
	
	// 检查每个路径分段
	pathSegments := strings.Split(strings.Trim(storagePath, "/"), "/")
	for _, segment := range pathSegments {
		if segment == "" {
			continue // 跳过空分段
		}
		
		// 检查保留名称（Windows系统保留名称）
		reservedNames := []string{
			"CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		}
		
		segmentUpper := strings.ToUpper(segment)
		for _, reserved := range reservedNames {
			if segmentUpper == reserved || strings.HasPrefix(segmentUpper, reserved+".") {
				return fmt.Errorf("storage path contains reserved name: %s", segment)
			}
		}
		
		// 检查分段长度
		if len(segment) > 255 {
			return fmt.Errorf("path segment too long: %s", segment)
		}
	}
	
	return nil
}

// SanitizeStoragePath 清理存储路径以确保安全性
func SanitizeStoragePath(storagePath string) string {
	// 首先标准化路径分隔符
	normalizedPath := strings.ReplaceAll(storagePath, "\\", "/")
	
	// 移除危险模式
	normalizedPath = strings.ReplaceAll(normalizedPath, "../", "")
	normalizedPath = strings.ReplaceAll(normalizedPath, "..\\", "")
	normalizedPath = strings.ReplaceAll(normalizedPath, "//", "/")
	normalizedPath = strings.ReplaceAll(normalizedPath, "./", "")
	
	// 分割路径并清理每个分段
	pathSegments := strings.Split(strings.Trim(normalizedPath, "/"), "/")
	cleanedSegments := make([]string, 0, len(pathSegments))
	
	for _, segment := range pathSegments {
		if segment == "" {
			continue
		}
		
		// 清理每个分段
		cleanedSegment := SanitizeFilename(segment)
		
		// 确保不是保留名称
		segmentUpper := strings.ToUpper(cleanedSegment)
		reservedNames := []string{
			"CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		}
		
		for _, reserved := range reservedNames {
			if segmentUpper == reserved {
				cleanedSegment = "safe_" + cleanedSegment
				break
			}
		}
		
		cleanedSegments = append(cleanedSegments, cleanedSegment)
	}
	
	result := strings.Join(cleanedSegments, "/")
	
	// 最终长度检查
	if len(result) > 1000 {
		result = result[:1000]
		// 确保不在路径分隔符处截断
		if lastSlash := strings.LastIndex(result, "/"); lastSlash > 0 {
			result = result[:lastSlash]
		}
	}
	
	return result
}