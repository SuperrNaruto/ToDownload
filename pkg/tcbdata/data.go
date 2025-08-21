package tcbdata

import (
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

const (
	TypeAdd                  = "add"
	TypeSetDefault           = "setdefault"
	TypeDeleteStorageConfirm = "delete_storage_confirm"
	TypeStorageToggle        = "storage_toggle"
)

// type TaskDataTGFiles struct {
// 	Files   []tfile.TGFileMessage
// 	AsBatch bool
// }

// type TaskDataTelegraph struct {
// 	Pics     []string
// 	PageNode *telegraph.Page
// }

// type TaskDataType interface {
// 	TaskDataTGFiles | TaskDataTelegraph
// }

type Add struct {
	TaskType         tasktype.TaskType
	SelectedStorName string
	DirID            uint
	SettedDir        bool
	// tfiles
	Files   []tfile.TGFileMessage
	AsBatch bool
	// tphpics
	TphPageNode *telegraph.Page
	TphPics     []string
	TphDirPath  string // unescaped telegraph.Page.Path
}

type SetDefaultStorage struct {
	StorageName string
}

// StorageConfigWizard 存储配置向导数据
type StorageConfigWizard struct {
	ChatID         int64
	StorageName    string
	StorageType    string
	Description    string
	ExpectedFields []string
}

// DeleteStorageConfirm 删除存储确认数据
type DeleteStorageConfirm struct {
	StorageID uint
	ChatID    int64
}

// StorageToggle 存储状态切换数据
type StorageToggle struct {
	StorageID uint
	ChatID    int64
}
