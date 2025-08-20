package configval

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid bool
	Error   string
	Suggestion string
}

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateAlistConfig 验证Alist配置
func (v *ConfigValidator) ValidateAlistConfig(input string) (*ValidationResult, map[string]string) {
	parts := strings.Split(input, ",")
	if len(parts) < 3 {
		return &ValidationResult{
			IsValid: false,
			Error: "配置信息不完整，至少需要URL、用户名和密码",
			Suggestion: "格式：https://alist.example.com,admin,password123,/upload",
		}, nil
	}

	config := make(map[string]string)
	config["url"] = strings.TrimSpace(parts[0])
	config["username"] = strings.TrimSpace(parts[1])
	config["password"] = strings.TrimSpace(parts[2])
	
	if len(parts) > 3 {
		config["base_path"] = strings.TrimSpace(parts[3])
	} else {
		config["base_path"] = "/"
	}

	// 验证URL格式
	if result := v.validateURL(config["url"]); !result.IsValid {
		return result, nil
	}

	// 验证用户名
	if config["username"] == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "用户名不能为空",
			Suggestion: "请提供有效的Alist用户名",
		}, nil
	}

	// 验证密码
	if config["password"] == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "密码不能为空", 
			Suggestion: "请提供有效的密码",
		}, nil
	}

	// 验证基础路径
	if !strings.HasPrefix(config["base_path"], "/") {
		config["base_path"] = "/" + config["base_path"]
	}

	return &ValidationResult{IsValid: true}, config
}

// ValidateWebDAVConfig 验证WebDAV配置
func (v *ConfigValidator) ValidateWebDAVConfig(input string) (*ValidationResult, map[string]string) {
	parts := strings.Split(input, ",")
	if len(parts) < 3 {
		return &ValidationResult{
			IsValid: false,
			Error: "配置信息不完整，至少需要URL、用户名和密码",
			Suggestion: "格式：https://webdav.example.com,user,pass123,/files",
		}, nil
	}

	config := make(map[string]string)
	config["url"] = strings.TrimSpace(parts[0])
	config["username"] = strings.TrimSpace(parts[1])
	config["password"] = strings.TrimSpace(parts[2])
	
	if len(parts) > 3 {
		config["path"] = strings.TrimSpace(parts[3])
	} else {
		config["path"] = "/"
	}

	// 验证URL格式
	if result := v.validateURL(config["url"]); !result.IsValid {
		return result, nil
	}

	// 验证路径
	if !strings.HasPrefix(config["path"], "/") {
		config["path"] = "/" + config["path"]
	}

	return &ValidationResult{IsValid: true}, config
}

// ValidateMinIOConfig 验证MinIO/S3配置
func (v *ConfigValidator) ValidateMinIOConfig(input string) (*ValidationResult, map[string]string) {
	parts := strings.Split(input, ",")
	if len(parts) < 4 {
		return &ValidationResult{
			IsValid: false,
			Error: "配置信息不完整，至少需要endpoint、access_key、secret_key和bucket",
			Suggestion: "格式：s3.amazonaws.com,KEY123,SECRET456,my-bucket,us-east-1",
		}, nil
	}

	config := make(map[string]string)
	config["endpoint"] = strings.TrimSpace(parts[0])
	config["access_key"] = strings.TrimSpace(parts[1])
	config["secret_key"] = strings.TrimSpace(parts[2])
	config["bucket"] = strings.TrimSpace(parts[3])
	
	if len(parts) > 4 {
		config["region"] = strings.TrimSpace(parts[4])
	} else {
		config["region"] = "us-east-1"
	}

	// 验证endpoint
	if config["endpoint"] == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "Endpoint不能为空",
			Suggestion: "请提供有效的S3端点地址，如：s3.amazonaws.com",
		}, nil
	}

	// 验证访问密钥
	if config["access_key"] == "" || len(config["access_key"]) < 16 {
		return &ValidationResult{
			IsValid: false,
			Error: "访问密钥格式不正确",
			Suggestion: "访问密钥通常长度不少于16位",
		}, nil
	}

	// 验证秘密密钥
	if config["secret_key"] == "" || len(config["secret_key"]) < 16 {
		return &ValidationResult{
			IsValid: false,
			Error: "秘密密钥格式不正确",
			Suggestion: "秘密密钥通常长度不少于16位",
		}, nil
	}

	// 验证bucket名称
	if !v.isValidBucketName(config["bucket"]) {
		return &ValidationResult{
			IsValid: false,
			Error: "存储桶名称格式不正确",
			Suggestion: "存储桶名称只能包含小写字母、数字和连字符，长度3-63位",
		}, nil
	}

	return &ValidationResult{IsValid: true}, config
}

// ValidateLocalConfig 验证本地存储配置
func (v *ConfigValidator) ValidateLocalConfig(input string) (*ValidationResult, map[string]string) {
	path := strings.TrimSpace(input)
	
	if path == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "路径不能为空",
			Suggestion: "请提供有效的绝对路径，如：/home/user/downloads",
		}, nil
	}

	// 验证是否为绝对路径
	if !strings.HasPrefix(path, "/") {
		return &ValidationResult{
			IsValid: false,
			Error: "必须使用绝对路径",
			Suggestion: "路径应以 / 开头，如：/home/user/downloads",
		}, nil
	}

	config := map[string]string{
		"base_path": path,
	}

	return &ValidationResult{IsValid: true}, config
}

// ValidateTelegramConfig 验证Telegram存储配置
func (v *ConfigValidator) ValidateTelegramConfig(input string) (*ValidationResult, map[string]string) {
	chatID := strings.TrimSpace(input)
	
	if chatID == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "频道ID不能为空",
			Suggestion: "请提供频道或群组的ID，如：-1001234567890",
		}, nil
	}

	// 验证是否为有效的chat_id格式
	if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
		return &ValidationResult{
			IsValid: false,
			Error: "频道ID格式不正确",
			Suggestion: "频道ID应为数字格式，如：-1001234567890",
		}, nil
	}

	// Telegram频道ID通常为负数且很长
	if !strings.HasPrefix(chatID, "-100") {
		return &ValidationResult{
			IsValid: false,
			Error: "频道ID格式可能不正确",
			Suggestion: "频道ID通常以-100开头，如：-1001234567890",
		}, nil
	}

	config := map[string]string{
		"chat_id": chatID,
	}

	return &ValidationResult{IsValid: true}, config
}

// validateURL 验证URL格式
func (v *ConfigValidator) validateURL(urlStr string) *ValidationResult {
	if urlStr == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "URL不能为空",
			Suggestion: "请提供有效的URL，如：https://example.com",
		}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return &ValidationResult{
			IsValid: false,
			Error: "URL格式无效",
			Suggestion: "URL格式应为：https://example.com",
		}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &ValidationResult{
			IsValid: false,
			Error: "URL协议必须是http或https",
			Suggestion: "请使用http://或https://开头的URL",
		}
	}

	if parsedURL.Host == "" {
		return &ValidationResult{
			IsValid: false,
			Error: "URL缺少主机名",
			Suggestion: "URL应包含主机名，如：https://example.com",
		}
	}

	return &ValidationResult{IsValid: true}
}

// isValidBucketName 验证S3存储桶名称
func (v *ConfigValidator) isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// 存储桶名称规则：只能包含小写字母、数字和连字符
	matched, _ := regexp.MatchString("^[a-z0-9][a-z0-9-]*[a-z0-9]$", name)
	if !matched && len(name) > 1 {
		return false
	}
	
	// 单字符存储桶名称只能是字母数字
	if len(name) == 1 {
		matched, _ := regexp.MatchString("^[a-z0-9]$", name)
		return matched
	}

	// 不能包含连续的连字符
	if strings.Contains(name, "--") {
		return false
	}

	return true
}

// GetSmartSuggestions 基于输入提供智能建议
func (v *ConfigValidator) GetSmartSuggestions(storageType, input string) []string {
	suggestions := []string{}

	switch storageType {
	case "alist":
		if strings.Contains(input, "http://") {
			suggestions = append(suggestions, "建议使用HTTPS协议以提高安全性")
		}
		if !strings.Contains(input, ",") {
			suggestions = append(suggestions, "配置项之间请用英文逗号分隔")
		}
	case "webdav":
		if strings.Contains(input, "dropbox") {
			suggestions = append(suggestions, "Dropbox不支持WebDAV，请使用其API接入")
		}
		if strings.Contains(input, "google") {
			suggestions = append(suggestions, "Google Drive不直接支持WebDAV")
		}
	case "minio":
		if strings.Contains(input, ".amazonaws.com") {
			suggestions = append(suggestions, "检测到AWS S3，region参数很重要")
		}
		if strings.Contains(input, "localhost") || strings.Contains(input, "127.0.0.1") {
			suggestions = append(suggestions, "本地MinIO服务，确保端口正确")
		}
	case "local":
		if strings.Contains(input, "C:\\") || strings.Contains(input, "D:\\") {
			suggestions = append(suggestions, "检测到Windows路径格式，服务器可能不支持")
		}
	case "telegram":
		if !strings.HasPrefix(input, "-") {
			suggestions = append(suggestions, "频道ID通常为负数")
		}
	}

	return suggestions
}

// FormatConfigPreview 格式化配置预览
func (v *ConfigValidator) FormatConfigPreview(config map[string]string, hideSecrets bool) map[string]string {
	preview := make(map[string]string)
	
	for key, value := range config {
		if hideSecrets && (key == "password" || key == "secret_key") {
			if len(value) > 6 {
				preview[key] = value[:3] + "****" + value[len(value)-3:]
			} else {
				preview[key] = "****"
			}
		} else {
			preview[key] = value
		}
	}
	
	return preview
}