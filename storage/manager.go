package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	storcfg "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/database"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

// StorageManager 存储管理器，整合系统配置存储和用户自定义存储
type StorageManager struct{}

// NewStorageManager 创建存储管理器
func NewStorageManager() *StorageManager {
	return &StorageManager{}
}

// GetUserStorageByName 获取用户存储（包括系统配置和自定义存储）
func (sm *StorageManager) GetUserStorageByName(ctx context.Context, chatID int64, storageName string) (Storage, error) {
	// 首先尝试从用户自定义存储中查找
	userStorage, err := sm.GetUserCustomStorageByName(ctx, chatID, storageName)
	if err == nil {
		return userStorage, nil
	}

	// 如果没找到，再从系统配置存储中查找
	return GetStorageByUserIDAndName(ctx, chatID, storageName)
}

// GetUserCustomStorageByName 获取用户自定义存储
func (sm *StorageManager) GetUserCustomStorageByName(ctx context.Context, chatID int64, storageName string) (Storage, error) {
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	userStorage, err := database.GetUserStorageByUserIDAndName(ctx, user.ID, storageName)
	if err != nil {
		return nil, fmt.Errorf("获取用户存储失败: %w", err)
	}

	if !userStorage.Enable {
		return nil, fmt.Errorf("存储 '%s' 已禁用", storageName)
	}

	// 将用户存储转换为存储配置
	storageConfig, err := sm.convertUserStorageToConfig(userStorage)
	if err != nil {
		return nil, fmt.Errorf("转换存储配置失败: %w", err)
	}

	// 创建存储实例
	storage, err := NewStorage(ctx, storageConfig)
	if err != nil {
		return nil, fmt.Errorf("创建存储实例失败: %w", err)
	}

	return storage, nil
}

// GetAllUserStorages 获取用户所有可用存储（系统配置 + 自定义存储）
func (sm *StorageManager) GetAllUserStorages(ctx context.Context, chatID int64) ([]Storage, error) {
	var allStorages []Storage

	// 获取系统配置存储
	systemStorages := GetUserStorages(ctx, chatID)
	allStorages = append(allStorages, systemStorages...)

	// 获取用户自定义存储
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return allStorages, nil // 如果用户不存在，只返回系统存储
	}

	userStorages, err := database.GetEnabledUserStoragesByUserID(ctx, user.ID)
	if err != nil {
		return allStorages, nil // 如果获取用户存储失败，只返回系统存储
	}

	for _, userStorage := range userStorages {
		storageConfig, err := sm.convertUserStorageToConfig(&userStorage)
		if err != nil {
			continue // 跳过转换失败的存储
		}

		storage, err := NewStorage(ctx, storageConfig)
		if err != nil {
			continue // 跳过创建失败的存储
		}

		allStorages = append(allStorages, storage)
	}

	return allStorages, nil
}

// GetUserStorageNames 获取用户所有存储名称
func (sm *StorageManager) GetUserStorageNames(ctx context.Context, chatID int64) ([]string, error) {
	storages, err := sm.GetAllUserStorages(ctx, chatID)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, storage := range storages {
		names = append(names, storage.Name())
	}

	return names, nil
}

// CreateUserStorage 创建用户自定义存储
func (sm *StorageManager) CreateUserStorage(ctx context.Context, chatID int64, name, storageType, description string, config map[string]interface{}) error {
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 验证存储类型
	if _, err := storenum.ParseStorageType(storageType); err != nil {
		return fmt.Errorf("不支持的存储类型: %s", storageType)
	}

	// 将配置转换为JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 验证配置有效性
	if err := database.ValidateStorageConfig(storageType, string(configJSON)); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 测试存储连接
	if err := sm.testStorageConnection(ctx, storageType, config); err != nil {
		return fmt.Errorf("存储连接测试失败: %w", err)
	}

	userStorage := &database.UserStorage{
		UserID:      user.ID,
		Name:        name,
		Type:        storageType,
		Enable:      true,
		Config:      string(configJSON),
		Description: description,
	}

	return database.CreateUserStorage(ctx, userStorage)
}

// UpdateUserStorage 更新用户自定义存储
func (sm *StorageManager) UpdateUserStorage(ctx context.Context, chatID int64, storageID uint, name, description string, config map[string]interface{}) error {
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	userStorage, err := database.GetUserStorageByID(ctx, storageID)
	if err != nil {
		return fmt.Errorf("获取存储配置失败: %w", err)
	}

	// 检查权限
	if userStorage.UserID != user.ID {
		return fmt.Errorf("无权限修改此存储配置")
	}

	// 将配置转换为JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 验证配置有效性
	if err := database.ValidateStorageConfig(userStorage.Type, string(configJSON)); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 测试存储连接
	if err := sm.testStorageConnection(ctx, userStorage.Type, config); err != nil {
		return fmt.Errorf("存储连接测试失败: %w", err)
	}

	userStorage.Name = name
	userStorage.Description = description
	userStorage.Config = string(configJSON)

	return database.UpdateUserStorage(ctx, userStorage)
}

// DeleteUserStorage 删除用户自定义存储
func (sm *StorageManager) DeleteUserStorage(ctx context.Context, chatID int64, storageID uint) error {
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	userStorage, err := database.GetUserStorageByID(ctx, storageID)
	if err != nil {
		return fmt.Errorf("获取存储配置失败: %w", err)
	}

	// 检查权限
	if userStorage.UserID != user.ID {
		return fmt.Errorf("无权限删除此存储配置")
	}

	// 检查是否为默认存储
	if user.DefaultStorage == userStorage.Name {
		return fmt.Errorf("无法删除默认存储，请先设置其他存储为默认")
	}

	return database.DeleteUserStorage(ctx, userStorage)
}

// ToggleUserStorageStatus 切换存储启用状态
func (sm *StorageManager) ToggleUserStorageStatus(ctx context.Context, chatID int64, storageID uint) error {
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	userStorage, err := database.GetUserStorageByID(ctx, storageID)
	if err != nil {
		return fmt.Errorf("获取存储配置失败: %w", err)
	}

	// 检查权限
	if userStorage.UserID != user.ID {
		return fmt.Errorf("无权限修改此存储配置")
	}

	// 如果要禁用的是默认存储，则拒绝
	if userStorage.Enable && user.DefaultStorage == userStorage.Name {
		return fmt.Errorf("无法禁用默认存储，请先设置其他存储为默认")
	}

	_, err = database.ToggleUserStorageStatus(ctx, storageID)
	return err
}

// convertUserStorageToConfig 将用户存储转换为存储配置接口
func (sm *StorageManager) convertUserStorageToConfig(userStorage *database.UserStorage) (storcfg.StorageConfig, error) {
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(userStorage.Config), &configData); err != nil {
		return nil, fmt.Errorf("解析存储配置失败: %w", err)
	}

	baseConfig := &storcfg.BaseConfig{
		Name:      userStorage.Name,
		Type:      userStorage.Type,
		Enable:    userStorage.Enable,
		RawConfig: configData,
	}

	storageType, err := storenum.ParseStorageType(userStorage.Type)
	if err != nil {
		return nil, fmt.Errorf("解析存储类型失败: %w", err)
	}

	// 使用存储工厂创建配置
	factory, ok := storcfg.GetStorageFactory(storageType)
	if !ok {
		return nil, fmt.Errorf("不支持的存储类型: %s", userStorage.Type)
	}

	return factory(baseConfig)
}

// testStorageConnection 测试存储连接
func (sm *StorageManager) testStorageConnection(ctx context.Context, storageType string, config map[string]interface{}) error {
	// 创建临时存储配置进行测试
	baseConfig := &storcfg.BaseConfig{
		Name:      "test_storage",
		Type:      storageType,
		Enable:    true,
		RawConfig: config,
	}

	storageTypeEnum, err := storenum.ParseStorageType(storageType)
	if err != nil {
		return fmt.Errorf("解析存储类型失败: %w", err)
	}

	factory, ok := storcfg.GetStorageFactory(storageTypeEnum)
	if !ok {
		return fmt.Errorf("不支持的存储类型: %s", storageType)
	}

	storageConfig, err := factory(baseConfig)
	if err != nil {
		return fmt.Errorf("创建存储配置失败: %w", err)
	}

	// 验证配置
	if err := storageConfig.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建存储实例
	storage, err := NewStorage(ctx, storageConfig)
	if err != nil {
		return fmt.Errorf("创建存储实例失败: %w", err)
	}

	// 测试连接 - 不同存储类型使用不同的测试方法
	switch storageType {
	case "telegram":
		// Telegram存储：检查chat_id是否有效
		testPath := storage.JoinStoragePath("/")
		storage.Exists(ctx, testPath) // 这会检查chat是否可访问
	case "local":
		// 本地存储：检查路径是否存在且可访问
		testPath := storage.JoinStoragePath("/")
		if !storage.Exists(ctx, testPath) {
			return fmt.Errorf("本地路径不存在或无访问权限")
		}
	default:
		// 对于网络存储（alist, webdav, minio），只要能创建存储实例就认为配置有效
		// 因为根路径可能不存在是正常的（比如空的存储桶）
		// 实际的连接测试会在首次使用时进行
	}

	return nil
}

// TestUserStorageConnection 测试用户存储连接
func (sm *StorageManager) TestUserStorageConnection(ctx context.Context, chatID int64, storageName string) error {
	user, err := database.GetUserByChatID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	userStorage, err := database.GetUserStorageByUserIDAndName(ctx, user.ID, storageName)
	if err != nil {
		return fmt.Errorf("获取存储配置失败: %w", err)
	}

	if !userStorage.Enable {
		return fmt.Errorf("存储已禁用")
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(userStorage.Config), &config); err != nil {
		return fmt.Errorf("解析存储配置失败: %w", err)
	}

	return sm.testStorageConnection(ctx, userStorage.Type, config)
}

// ValidateStorageConfigData 验证存储配置数据
func (sm *StorageManager) ValidateStorageConfigData(storageType string, configData []string) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	switch storageType {
	case "alist":
		if len(configData) < 3 {
			return nil, fmt.Errorf("Alist存储至少需要3个参数: URL,用户名,密码")
		}
		config["url"] = strings.TrimSpace(configData[0])
		config["username"] = strings.TrimSpace(configData[1])
		config["password"] = strings.TrimSpace(configData[2])
		if len(configData) >= 4 && strings.TrimSpace(configData[3]) != "" {
			config["path"] = strings.TrimSpace(configData[3])
		} else {
			config["path"] = "/"
		}

	case "webdav":
		if len(configData) < 3 {
			return nil, fmt.Errorf("WebDAV存储至少需要3个参数: URL,用户名,密码")
		}
		config["url"] = strings.TrimSpace(configData[0])
		config["username"] = strings.TrimSpace(configData[1])
		config["password"] = strings.TrimSpace(configData[2])
		if len(configData) >= 4 && strings.TrimSpace(configData[3]) != "" {
			config["path"] = strings.TrimSpace(configData[3])
		} else {
			config["path"] = "/"
		}

	case "minio":
		if len(configData) < 4 {
			return nil, fmt.Errorf("MinIO/S3存储至少需要4个参数: endpoint,access_key,secret_key,bucket")
		}
		config["endpoint"] = strings.TrimSpace(configData[0])
		config["access_key"] = strings.TrimSpace(configData[1])
		config["secret_key"] = strings.TrimSpace(configData[2])
		config["bucket"] = strings.TrimSpace(configData[3])
		if len(configData) >= 5 && strings.TrimSpace(configData[4]) != "" {
			config["region"] = strings.TrimSpace(configData[4])
		} else {
			config["region"] = "us-east-1"
		}
		config["use_ssl"] = true

	case "local":
		if len(configData) < 1 {
			return nil, fmt.Errorf("本地存储需要1个参数: 路径")
		}
		config["base_path"] = strings.TrimSpace(configData[0])

	case "telegram":
		if len(configData) < 1 {
			return nil, fmt.Errorf("Telegram存储需要1个参数: chat_id")
		}
		chatIDStr := strings.TrimSpace(configData[0])
		chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的chat_id格式: %s", chatIDStr)
		}
		config["chat_id"] = chatID

	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", storageType)
	}

	return config, nil
}

// 全局存储管理器实例
var Manager = NewStorageManager()
