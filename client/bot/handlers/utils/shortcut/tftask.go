package shortcut

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/msgelem"
	"github.com/krau/SaveAny-Bot/client/bot/handlers/utils/ruleutil"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/batchtftask"
	"github.com/krau/SaveAny-Bot/core/tftask"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/consts"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// 创建一个 tftask.TGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddTGFileTaskWithEdit(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, file tfile.TGFileMessage, trackMsgID int) error {
	logger := log.FromContext(ctx)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		logger.Errorf("Failed to get user by chat ID: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "获取用户失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	if user.ApplyRule && user.Rules != nil {
		matchedStorageName, matchedDirPath := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))
		dirPath = matchedDirPath.String()
		if matchedStorageName.IsUsable() {
			stor, err = storage.Manager.GetUserStorageByName(ctx, user.ChatID, matchedStorageName.String())
			if err != nil {
				logger.Errorf("Failed to get storage by user ID and name: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "获取存储失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
		}
	}

	// Generate filename using AI if available, otherwise use original
	fileName := tgutil.GenFileNameFromMessage(*file.Message())
	storagePath := stor.JoinStoragePath(path.Join(dirPath, fileName))

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task, err := tftask.NewTGFileTask(taskid, injectCtx, file, stor, storagePath,
		tftask.NewProgressTrack(
			trackMsgID,
			userID))
	if err == nil {
		// Set the custom filename for display purposes
		task.SetCustomName(fileName)
	}
	if err != nil {
		logger.Errorf("create task failed: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "创建任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("add task failed: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "添加任务失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	text, entities := msgelem.BuildTaskAddedEntities(ctx, fileName, core.GetLength(injectCtx))
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:       trackMsgID,
		Message:  text,
		Entities: entities,
	})

	return dispatcher.EndGroups
}

// 创建一个 batchtftask.BatchTGFileTask 并添加到任务队列中, 以编辑消息的方式反馈结果
func CreateAndAddBatchTGFileTaskWithEdit(ctx *ext.Context, userID int64, stor storage.Storage, dirPath string, files []tfile.TGFileMessage, trackMsgID int) error {
	logger := log.FromContext(ctx)
	user, err := database.GetUserByChatID(ctx, userID)
	if err != nil {
		logger.Errorf("Failed to get user by chat ID: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "获取用户失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}

	useRule := user.ApplyRule && user.Rules != nil

	applyRule := func(file tfile.TGFileMessage) (string, ruleutil.MatchedDirPath) {
		if !useRule {
			return stor.Name(), ruleutil.MatchedDirPath(dirPath)
		}
		storName, dirP := ruleutil.ApplyRule(ctx, user.Rules, ruleutil.NewInput(file))

		storname := storName.String()
		if !storName.IsUsable() {
			storname = stor.Name()
		}
		return storname, dirP
	}

	elems := make([]batchtftask.TaskElement, 0, len(files))
	type albumFile struct {
		file    tfile.TGFileMessage
		storage storage.Storage
	}
	albumFiles := make(map[int64][]albumFile, 0)
	for _, file := range files {
		storName, dirPath := applyRule(file)
		fileStor := stor
		if storName != stor.Name() && storName != "" {
			fileStor, err = storage.Manager.GetUserStorageByName(ctx, user.ChatID, storName)
			if err != nil {
				logger.Errorf("Failed to get storage by user ID and name: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "获取存储失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
		}
		if !dirPath.NeedNewForAlbum() {
			storPath := fileStor.JoinStoragePath(path.Join(dirPath.String(), file.Name()))
			elem, err := batchtftask.NewTaskElement(fileStor, storPath, file)
			if err != nil {
				logger.Errorf("Failed to create task element: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "任务创建失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
			elems = append(elems, *elem)
		} else {
			groupId, isGroup := file.Message().GetGroupedID()
			if !isGroup || groupId == 0 {
				logger.Warnf("File %s is not in a group, skipping album handling", file.Name())
				continue
			}
			if _, ok := albumFiles[groupId]; !ok {
				albumFiles[groupId] = make([]albumFile, 0)
			}
			albumFiles[groupId] = append(albumFiles[groupId], albumFile{
				file:    file,
				storage: fileStor,
			})
		}
	}
	for _, afiles := range albumFiles {
		if len(afiles) <= 1 {
			continue
		}

		// 提取tfile.TGFileMessage切片用于AI重命名
		albumTFiles := make([]tfile.TGFileMessage, len(afiles))
		for i, af := range afiles {
			albumTFiles[i] = af.file
		}

		// 生成相册文件名（这将统一使用一次AI调用）
		albumFilenames, err := tgutil.GenerateAlbumFilenames(ctx, albumTFiles)
		if err != nil {
			logger.Errorf("Failed to generate album filenames: %s", err)
			// 如果AI重命名失败，使用原有文件名
			albumFilenames = make([]string, len(afiles))
			for i, af := range afiles {
				albumFilenames[i] = af.file.Name()
			}
		}

		// 从第一个文件名中提取基础名称作为文件夹名（确保命名一致）
		var albumDir string
		if len(albumFilenames) > 0 {
			// 提取基础名称：去掉序号和扩展名
			firstFilename := albumFilenames[0]
			ext := filepath.Ext(firstFilename)
			nameWithoutExt := strings.TrimSuffix(firstFilename, ext)

			// 找到最后一个下划线的位置，去掉序号部分（如 _01）
			if lastUnderscoreIndex := strings.LastIndex(nameWithoutExt, "_"); lastUnderscoreIndex > 0 {
				potentialSequence := nameWithoutExt[lastUnderscoreIndex+1:]
				// 检查是否为数字序号（2-3位数字）
				var num int
				if n, err := fmt.Sscanf(potentialSequence, "%d", &num); err == nil && n == 1 && len(potentialSequence) <= 3 {
					albumDir = nameWithoutExt[:lastUnderscoreIndex]
				} else {
					albumDir = nameWithoutExt
				}
			} else {
				albumDir = nameWithoutExt
			}
		} else {
			// 回退到原有逻辑
			albumDir = strings.TrimSuffix(path.Base(afiles[0].file.Name()), path.Ext(afiles[0].file.Name()))
		}

		// 存储以第一个文件的存储为准
		albumStor := afiles[0].storage

		// 获取第一个文件的目录路径（所有相册文件应该使用相同的目录路径）
		firstDirPath := ""
		if len(afiles) > 0 {
			_, firstFileDirPath := applyRule(afiles[0].file)
			firstDirPath = firstFileDirPath.String()
		}

		// 如果dirPath是NEW-FOR-ALBUM，则直接使用albumDir作为目录
		var finalDirPath string
		if firstDirPath == consts.RuleDirPathNewForAlbum {
			finalDirPath = albumDir
		} else {
			finalDirPath = path.Join(firstDirPath, albumDir)
		}

		for i, af := range afiles {
			// 使用生成的文件名而不是原文件名
			afstorPath := af.storage.JoinStoragePath(path.Join(finalDirPath, albumFilenames[i]))
			elem, err := batchtftask.NewTaskElement(albumStor, afstorPath, af.file)
			if err != nil {
				logger.Errorf("Failed to create task element for album file: %s", err)
				ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
					ID:      trackMsgID,
					Message: "任务创建失败: " + err.Error(),
				})
				return dispatcher.EndGroups
			}
			elems = append(elems, *elem)
		}
	}

	injectCtx := tgutil.ExtWithContext(ctx.Context, ctx)
	taskid := xid.New().String()
	task := batchtftask.NewBatchTGFileTask(taskid, injectCtx, elems, batchtftask.NewProgressTracker(trackMsgID, userID), true)
	if err := core.AddTask(injectCtx, task); err != nil {
		logger.Errorf("Failed to add batch task: %s", err)
		ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
			ID:      trackMsgID,
			Message: "批量任务添加失败: " + err.Error(),
		})
		return dispatcher.EndGroups
	}
	ctx.EditMessage(userID, &tg.MessagesEditMessageRequest{
		ID:          trackMsgID,
		Message:     fmt.Sprintf("已添加批量任务, 共 %d 个文件", len(files)),
		ReplyMarkup: nil,
	})
	return dispatcher.EndGroups
}
