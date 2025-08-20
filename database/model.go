package database

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ChatID         int64 `gorm:"uniqueIndex;not null"`
	Silent         bool
	DefaultStorage string
	Dirs           []Dir
	ApplyRule      bool
	Rules          []Rule
	WatchChats     []WatchChat
	UserStorages   []UserStorage
}

type WatchChat struct {
	gorm.Model
	UserID uint // User's database ID (not chat ID)
	ChatID int64
	Filter string
}

type Dir struct {
	gorm.Model
	UserID      uint
	StorageName string
	Path        string
}

type Rule struct {
	gorm.Model
	UserID      uint
	Type        string
	Data        string
	StorageName string
	DirPath     string
}

// UserStorage 用户自定义存储配置
type UserStorage struct {
	gorm.Model
	UserID      uint   `gorm:"not null;uniqueIndex:idx_user_storage_name"`
	Name        string `gorm:"not null;uniqueIndex:idx_user_storage_name"` // 存储名称
	Type        string `gorm:"not null"`                                   // 存储类型 (alist, webdav, minio, local, telegram)
	Enable      bool   `gorm:"default:true"`                               // 是否启用
	Config      string `gorm:"type:text"`                                  // JSON格式的配置数据
	Description string `gorm:"size:200"`                                   // 描述信息
	User        User   `gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (UserStorage) TableName() string {
	return "user_storages"
}
