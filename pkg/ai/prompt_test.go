package ai

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name     string
		req      RenameRequest
		expected string
	}{
		{
			name: "Normal file request",
			req: RenameRequest{
				OriginalFilename: "IMG_20240101_120000.jpg",
				MessageContent:   "这是一张风景照片，拍摄于北京香山",
				IsAlbum:          false,
			},
			expected: "IMG_20240101_120000.jpg",
		},
		{
			name: "Album request",
			req: RenameRequest{
				OriginalFilename: "",
				MessageContent:   "今天去香山拍的风景照片集",
				IsAlbum:          true,
			},
			expected: AlbumRenamePrompt,
		},
		{
			name: "Long message content truncation",
			req: RenameRequest{
				OriginalFilename: "test.txt",
				MessageContent:   strings.Repeat("很长的消息内容", 100), // Creates a very long message
				IsAlbum:          false,
			},
			expected: "test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPrompt(tt.req)
			
			// For normal file request, check that original filename is included
			if !tt.req.IsAlbum && !strings.Contains(result, tt.req.OriginalFilename) {
				t.Errorf("BuildPrompt() should contain original filename %s", tt.req.OriginalFilename)
			}
			
			// For album request, should use album prompt template
			if tt.req.IsAlbum && !strings.Contains(result, "相册") {
				t.Errorf("BuildPrompt() should use album template for album requests")
			}
			
			// Check message content truncation for long messages
			if len(tt.req.MessageContent) > 1000 {
				if strings.Contains(result, strings.Repeat("很长的消息内容", 100)) {
					t.Errorf("BuildPrompt() should truncate long message content")
				}
			}
		})
	}
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "Valid filename",
			filename: "北京香山.张三.2024年1月.风景照",
			expected: true,
		},
		{
			name:     "Empty filename",
			filename: "",
			expected: false,
		},
		{
			name:     "Too long filename",
			filename: strings.Repeat("很长的文件名", 20),
			expected: false,
		},
		{
			name:     "Contains forward slash",
			filename: "文件/名称",
			expected: false,
		},
		{
			name:     "Contains backslash",
			filename: "文件\\名称",
			expected: false,
		},
		{
			name:     "Contains colon",
			filename: "文件:名称",
			expected: false,
		},
		{
			name:     "Contains asterisk",
			filename: "文件*名称",
			expected: false,
		},
		{
			name:     "Contains question mark",
			filename: "文件?名称",
			expected: false,
		},
		{
			name:     "Contains quote",
			filename: "文件\"名称",
			expected: false,
		},
		{
			name:     "Contains less than",
			filename: "文件<名称",
			expected: false,
		},
		{
			name:     "Contains greater than",
			filename: "文件>名称",
			expected: false,
		},
		{
			name:     "Contains pipe",
			filename: "文件|名称",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("ValidateFilename(%s) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "Valid filename unchanged",
			filename: "北京香山.张三.2024年1月.风景照",
			expected: "北京香山.张三.2024年1月.风景照",
		},
		{
			name:     "Replace invalid characters",
			filename: "文件/名\\称:测*试?文\"件<名>称|测试",
			expected: "文件_名_称_测_试_文_件_名_称_测试",
		},
		{
			name:     "Remove multiple underscores",
			filename: "文件____名称",
			expected: "文件_名称",
		},
		{
			name:     "Trim leading and trailing underscores",
			filename: "_文件名称_",
			expected: "文件名称",
		},
		{
			name:     "Trim leading and trailing whitespace",
			filename: "  文件名称  ",
			expected: "文件名称",
		},
		{
			name:     "Truncate long filename",
			filename: strings.Repeat("很长的文件名", 20),
			expected: strings.Repeat("很长的文件名", 20)[:200],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%s) = %s, expected %s", tt.filename, result, tt.expected)
			}
			
			// Ensure result is always valid
			if !ValidateFilename(result) && result != "" {
				t.Errorf("SanitizeFilename(%s) produced invalid filename: %s", tt.filename, result)
			}
		})
	}
}