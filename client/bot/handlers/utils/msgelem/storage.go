package msgelem

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/gotd/td/telegram/message/entity"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

func BuildAddSelectStorageKeyboard(ctx context.Context, chatID int64, adddata tcbdata.Add) (*tg.ReplyInlineMarkup, error) {
	// 获取所有可用存储（系统配置 + 用户自定义）
	stors, err := storage.Manager.GetAllUserStorages(ctx, chatID)
	if err != nil {
		// 如果获取失败，回退到系统存储
		stors = storage.GetUserStorages(ctx, chatID)
	}
	taskType := adddata.TaskType
	if taskType == "" {
		if len(adddata.Files) > 0 {
			taskType = tasktype.TaskTypeTgfiles
		} else if adddata.TphPageNode != nil {
			taskType = tasktype.TaskTypeTphpics
		} else {
			return nil, fmt.Errorf("unknown task type: %s", taskType)
		}
	}

	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range stors {
		data := tcbdata.Add{
			TaskType:         taskType,
			SelectedStorName: storage.Name(),

			Files:   adddata.Files,
			AsBatch: len(adddata.Files) > 1,

			TphPageNode: adddata.TphPageNode,
			TphPics:     adddata.TphPics,
			TphDirPath:  adddata.TphDirPath,
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dataid),
		})
	}
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}

func BuildAddOneSelectStorageMessage(ctx context.Context, chatID int64, file tfile.TGFileMessage, msgId int) (*tg.MessagesEditMessageRequest, error) {
	eb := entity.Builder{}
	var entities []tg.MessageEntityClass
	text := fmt.Sprintf("文件名: %s\n请选择存储位置", file.Name())
	if err := styling.Perform(&eb,
		styling.Plain("文件名: "),
		styling.Code(file.Name()),
		styling.Plain("\n请选择存储位置"),
	); err != nil {
		log.FromContext(ctx).Errorf("Failed to build entity: %s", err)
	} else {
		text, entities = eb.Complete()
	}
	markup, err := BuildAddSelectStorageKeyboard(ctx, chatID, tcbdata.Add{
		TaskType: tasktype.TaskTypeTgfiles,
		Files:    []tfile.TGFileMessage{file},
		AsBatch:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build storage keyboard: %w", err)
	}
	return &tg.MessagesEditMessageRequest{
		Message:     text,
		Entities:    entities,
		ReplyMarkup: markup,
		ID:          msgId,
	}, nil
}

func BuildSetDefaultStorageMarkup(ctx context.Context, userID int64) (*tg.ReplyInlineMarkup, error) {
	// 获取所有可用存储（系统配置 + 用户自定义）
	stors, err := storage.Manager.GetAllUserStorages(ctx, userID)
	if err != nil {
		// 如果获取失败，回退到系统存储
		stors = storage.GetUserStorages(ctx, userID)
	}
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, storage := range stors {
		data := tcbdata.SetDefaultStorage{
			StorageName: storage.Name(),
		}
		dataid := xid.New().String()
		err := cache.Set(dataid, data)
		if err != nil {
			return nil, err
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: storage.Name(),
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeSetDefault, dataid),
		})
	}
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}

func BuildSetDirKeyboard(dirs []database.Dir, dataid string) (*tg.ReplyInlineMarkup, error) {
	data, ok := cache.Get[tcbdata.Add](dataid)
	if !ok {
		return nil, fmt.Errorf("failed to get data from cache: %s", dataid)
	}
	if data.DirID != 0 || data.SettedDir {
		log.Warnf("Data already has a directory set: %d, %t", data.DirID, data.SettedDir)
		return nil, fmt.Errorf("data already has a directory set")
	}
	buttons := make([]tg.KeyboardButtonClass, 0)
	for _, dir := range dirs {
		dirDataId := xid.New().String()
		dirData := data
		dirData.DirID = dir.ID
		dirData.SettedDir = true
		err := cache.Set(dirDataId, dirData)
		if err != nil {
			return nil, fmt.Errorf("failed to set directory data in cache: %w", err)
		}
		buttons = append(buttons, &tg.KeyboardButtonCallback{
			Text: dir.Path,
			Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dirDataId),
		})
	}
	dirDefaultDataId := xid.New().String()
	dirDefaultData := data
	dirDefaultData.DirID = 0
	dirDefaultData.SettedDir = true
	err := cache.Set(dirDefaultDataId, dirDefaultData)
	if err != nil {
		return nil, fmt.Errorf("failed to set default directory data in cache: %w", err)
	}
	buttons = append(buttons, &tg.KeyboardButtonCallback{
		Text: "默认",
		Data: fmt.Appendf(nil, "%s %s", tcbdata.TypeAdd, dirDefaultDataId),
	})
	markup := &tg.ReplyInlineMarkup{}
	for i := 0; i < len(buttons); i += 3 {
		row := tg.KeyboardButtonRow{}
		row.Buttons = buttons[i:min(i+3, len(buttons))]
		markup.Rows = append(markup.Rows, row)
	}
	return markup, nil
}

// BuildStorageManageMarkup 构建存储管理按钮
func BuildStorageManageMarkup(ctx context.Context, userStorages []database.UserStorage) (*tg.ReplyInlineMarkup, error) {
	var rows []tg.KeyboardButtonRow

	// 为每个存储添加操作按钮
	for _, storage := range userStorages {
		var statusIcon string
		var toggleText string
		if storage.Enable {
			statusIcon = "🟢"
			toggleText = "禁用"
		} else {
			statusIcon = "🔴"
			toggleText = "启用"
		}

		row := tg.KeyboardButtonRow{
			Buttons: []tg.KeyboardButtonClass{
				&tg.KeyboardButtonCallback{
					Text: fmt.Sprintf("%s %s", statusIcon, storage.Name),
					Data: []byte(fmt.Sprintf("storage_info %d", storage.ID)),
				},
				&tg.KeyboardButtonCallback{
					Text: toggleText,
					Data: []byte(fmt.Sprintf("%s %d", tcbdata.TypeStorageToggle, storage.ID)),
				},
			},
		}
		rows = append(rows, row)
	}

	// 添加通用操作按钮
	actionRow := tg.KeyboardButtonRow{
		Buttons: []tg.KeyboardButtonClass{
			&tg.KeyboardButtonCallback{
				Text: "➕ 添加存储",
				Data: []byte("storage_add_start"),
			},
		},
	}
	rows = append(rows, actionRow)

	return &tg.ReplyInlineMarkup{Rows: rows}, nil
}

// BuildStorageTypeSelectMarkup 构建存储类型选择按钮
func BuildStorageTypeSelectMarkup() *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📁 Alist",
						Data: []byte("storage_type_alist"),
					},
					&tg.KeyboardButtonCallback{
						Text: "🌐 WebDAV",
						Data: []byte("storage_type_webdav"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "☁️ MinIO/S3",
						Data: []byte("storage_type_minio"),
					},
					&tg.KeyboardButtonCallback{
						Text: "💾 本地存储",
						Data: []byte("storage_type_local"),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "📱 Telegram",
						Data: []byte("storage_type_telegram"),
					},
					&tg.KeyboardButtonCallback{
						Text: "❌ 取消",
						Data: []byte("cancel"),
					},
				},
			},
		},
	}
}

// BuildStorageDetailMarkup 构建存储详情操作按钮
func BuildStorageDetailMarkup(storageID uint) *tg.ReplyInlineMarkup {
	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "✏️ 编辑",
						Data: []byte(fmt.Sprintf("storage_edit %d", storageID)),
					},
					&tg.KeyboardButtonCallback{
						Text: "🧪 测试",
						Data: []byte(fmt.Sprintf("storage_test %d", storageID)),
					},
				},
			},
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonCallback{
						Text: "🗑️ 删除",
						Data: []byte(fmt.Sprintf("storage_delete %d", storageID)),
					},
					&tg.KeyboardButtonCallback{
						Text: "⬅️ 返回",
						Data: []byte("storage_list_refresh"),
					},
				},
			},
		},
	}
}
