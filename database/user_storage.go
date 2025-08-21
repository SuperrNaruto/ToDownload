package database

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

// CreateUserStorage 创建用户存储配置
func CreateUserStorage(ctx context.Context, userStorage *UserStorage) error {
	// 检查存储名称是否已存在
	var existing UserStorage
	if err := db.WithContext(ctx).Where("user_id = ? AND name = ?", userStorage.UserID, userStorage.Name).First(&existing).Error; err == nil {
		return fmt.Errorf("存储名称 '%s' 已存在", userStorage.Name)
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查存储名称失败: %w", err)
	}

	return db.WithContext(ctx).Create(userStorage).Error
}

// GetUserStorageByID 根据ID获取用户存储配置
func GetUserStorageByID(ctx context.Context, id uint) (*UserStorage, error) {
	var userStorage UserStorage
	err := db.WithContext(ctx).
		Preload("User").
		Where("id = ?", id).First(&userStorage).Error
	if err != nil {
		return nil, err
	}
	return &userStorage, nil
}

// GetUserStorageByUserIDAndName 根据用户ID和存储名称获取存储配置
func GetUserStorageByUserIDAndName(ctx context.Context, userID uint, name string) (*UserStorage, error) {
	var userStorage UserStorage
	err := db.WithContext(ctx).
		Preload("User").
		Where("user_id = ? AND name = ?", userID, name).First(&userStorage).Error
	if err != nil {
		return nil, err
	}
	return &userStorage, nil
}

// GetUserStoragesByUserID 获取用户的所有存储配置
func GetUserStoragesByUserID(ctx context.Context, userID uint) ([]UserStorage, error) {
	var userStorages []UserStorage
	err := db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&userStorages).Error
	return userStorages, err
}

// GetUserStoragesByChatID 根据聊天ID获取用户的所有存储配置
func GetUserStoragesByChatID(ctx context.Context, chatID int64) ([]UserStorage, error) {
	user, err := GetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	return GetUserStoragesByUserID(ctx, user.ID)
}

// GetEnabledUserStoragesByUserID 获取用户的所有启用的存储配置
func GetEnabledUserStoragesByUserID(ctx context.Context, userID uint) ([]UserStorage, error) {
	var userStorages []UserStorage
	err := db.WithContext(ctx).
		Where("user_id = ? AND enable = ?", userID, true).
		Order("created_at DESC").
		Find(&userStorages).Error
	return userStorages, err
}

// UpdateUserStorage 更新用户存储配置
func UpdateUserStorage(ctx context.Context, userStorage *UserStorage) error {
	// 检查存储是否存在
	if _, err := GetUserStorageByID(ctx, userStorage.ID); err != nil {
		return fmt.Errorf("存储配置不存在: %w", err)
	}

	// 检查名称冲突（排除自身）
	var existing UserStorage
	if err := db.WithContext(ctx).Where("user_id = ? AND name = ? AND id != ?",
		userStorage.UserID, userStorage.Name, userStorage.ID).First(&existing).Error; err == nil {
		return fmt.Errorf("存储名称 '%s' 已存在", userStorage.Name)
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查存储名称失败: %w", err)
	}

	return db.WithContext(ctx).Save(userStorage).Error
}

// DeleteUserStorage 删除用户存储配置
func DeleteUserStorage(ctx context.Context, userStorage *UserStorage) error {
	return db.WithContext(ctx).Delete(userStorage).Error
}

// DeleteUserStorageByID 根据ID删除用户存储配置
func DeleteUserStorageByID(ctx context.Context, id uint) error {
	return db.WithContext(ctx).Delete(&UserStorage{}, id).Error
}

// ToggleUserStorageStatus 切换存储启用状态
func ToggleUserStorageStatus(ctx context.Context, id uint) (*UserStorage, error) {
	var userStorage UserStorage
	if err := db.WithContext(ctx).Where("id = ?", id).First(&userStorage).Error; err != nil {
		return nil, fmt.Errorf("存储配置不存在: %w", err)
	}

	userStorage.Enable = !userStorage.Enable
	if err := db.WithContext(ctx).Save(&userStorage).Error; err != nil {
		return nil, fmt.Errorf("更新存储状态失败: %w", err)
	}

	return &userStorage, nil
}

// CountUserStorages 统计用户存储配置数量
func CountUserStorages(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&UserStorage{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// ValidateStorageConfig 验证存储配置JSON的有效性
func ValidateStorageConfig(storageType, configJSON string) error {
	if configJSON == "" {
		return fmt.Errorf("配置不能为空")
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("配置格式无效: %w", err)
	}

	// 根据存储类型验证必需字段
	switch storageType {
	case "alist":
		if _, ok := config["url"]; !ok {
			return fmt.Errorf("Alist存储缺少必需字段: url")
		}
		if _, ok := config["username"]; !ok {
			return fmt.Errorf("Alist存储缺少必需字段: username")
		}
		if _, ok := config["password"]; !ok {
			return fmt.Errorf("Alist存储缺少必需字段: password")
		}
		if _, ok := config["base_path"]; !ok {
			return fmt.Errorf("Alist存储缺少必需字段: base_path")
		}
	case "webdav":
		if _, ok := config["url"]; !ok {
			return fmt.Errorf("WebDAV存储缺少必需字段: url")
		}
		if _, ok := config["username"]; !ok {
			return fmt.Errorf("WebDAV存储缺少必需字段: username")
		}
		if _, ok := config["password"]; !ok {
			return fmt.Errorf("WebDAV存储缺少必需字段: password")
		}
	case "minio":
		requiredFields := []string{"endpoint", "access_key", "secret_key", "bucket"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				return fmt.Errorf("MinIO/S3存储缺少必需字段: %s", field)
			}
		}
	case "local":
		if _, ok := config["base_path"]; !ok {
			return fmt.Errorf("本地存储缺少必需字段: base_path")
		}
	case "telegram":
		if _, ok := config["chat_id"]; !ok {
			return fmt.Errorf("Telegram存储缺少必需字段: chat_id")
		}
	default:
		return fmt.Errorf("不支持的存储类型: %s", storageType)
	}

	return nil
}

// GetUserStorageWithConfig 获取带完整配置的用户存储
func GetUserStorageWithConfig(ctx context.Context, userID uint, name string) (*UserStorage, map[string]interface{}, error) {
	userStorage, err := GetUserStorageByUserIDAndName(ctx, userID, name)
	if err != nil {
		return nil, nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(userStorage.Config), &config); err != nil {
		return userStorage, nil, fmt.Errorf("解析存储配置失败: %w", err)
	}

	return userStorage, config, nil
}
