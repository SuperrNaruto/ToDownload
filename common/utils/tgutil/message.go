package tgutil

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/celestix/gotgproto/ext"
	"github.com/charmbracelet/log"
	"github.com/duke-git/lancet/v2/maputil"

	"github.com/duke-git/lancet/v2/mathutil"
	"github.com/duke-git/lancet/v2/slice"
	lcstrutil "github.com/duke-git/lancet/v2/strutil"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gotd/td/tg"
	"github.com/krau/SaveAny-Bot/common/cache"
	"github.com/krau/SaveAny-Bot/common/utils/strutil"
	"github.com/krau/SaveAny-Bot/pkg/ai"
	"github.com/rs/xid"
)

// generate a file name from the message content and media type
//
// it will never return an empty string
func GenFileNameFromMessage(message tg.Message) string {
	return GenFileNameFromMessageWithContext(context.Background(), message)
}

// GenFileNameFromMessageWithContext generates a file name using AI when available
func GenFileNameFromMessageWithContext(ctx context.Context, message tg.Message) string {
	logger := log.FromContext(ctx)
	
	// Extract file extension from original filename when available
	ext := func(media tg.MessageMediaClass) string {
		// First try to get extension from original filename
		if originalFilename, err := GetMediaFileName(media); err == nil && originalFilename != "" {
			if extIdx := strings.LastIndex(originalFilename, "."); extIdx != -1 {
				fileExt := originalFilename[extIdx:]
				logger.Debugf("Using original file extension: %s from filename: %s", fileExt, originalFilename)
				return fileExt
			}
		}
		
		// Fallback to media type based extension
		switch media := media.(type) {
		case *tg.MessageMediaDocument:
			doc, ok := media.Document.AsNotEmpty()
			if !ok {
				return ""
			}
			mmt := mimetype.Lookup(doc.MimeType)
			if mmt == nil || mmt.Extension() == "" {
				return ""
			}
			logger.Debugf("Using mimetype extension: %s for document", mmt.Extension())
			return mmt.Extension()
		case *tg.MessageMediaPhoto:
			logger.Debugf("Using default .jpg extension for photo")
			return ".jpg"
		}
		return ""
	}(message.Media)

	text := strings.TrimSpace(message.GetMessage())
	
	// Try AI renaming if service is available and initialized
	if IsRenameServiceInitialized() {
		renameService := GetRenameService()
		if renameService != nil && renameService.IsEnabled() {
			// Get original filename from media if available
			originalFilename := ""
			if mname, err := GetMediaFileName(message.Media); err == nil && mname != "" {
				originalFilename = mname
			}
			
			logger.Debugf("AI rename: originalFilename=%s, ext=%s, text=%s", originalFilename, ext, text)
			
			// Create a timeout context for AI call
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			
			// Call AI rename service
			aiFilename, err := renameService.RenameFile(ctxWithTimeout, originalFilename, text)
			if err == nil && aiFilename != "" {
				// 额外的安全验证，确保AI生成的文件名不会导致路径问题
				if strings.ContainsAny(aiFilename, "/\\:*?\"<>|") {
					logger.Warnf("AI generated filename contains unsafe characters: %s, sanitizing", aiFilename)
					// 使用AI模块的清理函数
					aiFilename = ai.SanitizeFilename(aiFilename)
				}
				
				finalName := aiFilename + ext
				logger.Debugf("AI rename successful: %s -> %s", originalFilename, finalName)
				return finalName
			}
			logger.Debugf("AI rename failed or empty result, using fallback")
			// If AI fails, continue with fallback logic below
		}
	}
	
	// Fallback to original logic if AI is not available or fails
	return genFileNameFromMessageOriginal(message, ext, text)
}

// genFileNameFromMessageOriginal contains the original file naming logic as fallback
func genFileNameFromMessageOriginal(message tg.Message, ext, text string) string {
	if text == "" {
		return fmt.Sprintf("%d_%s%s", message.GetID(), xid.New().String(), ext)
	}
	filename := func() string {
		tags := strutil.ExtractTagsFromText(text)
		if len(tags) > 0 {
			tagStrRunes := make([]rune, 0, 64)
			for i, tag := range tags {
				if i > 0 {
					tagStrRunes = append(tagStrRunes, '_')
				}
				tagStrRunes = append(tagStrRunes, []rune(tag)...)
				if len(tagStrRunes) >= 64 {
					break
				}
			}
			tagStr := string(tagStrRunes)
			return fmt.Sprintf("%s_%s", tagStr, strconv.Itoa(message.GetID()))
		}
		text = lcstrutil.Substring(strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7F {
				return '_'
			}
			switch r {
			// invalid characters
			case '/', '\\',
				':', '*', '?', '"', '<', '>', '|':
				return '_'
			// empty
			case ' ', '\t', '\r', '\n':
				return '_'
			}
			if validator.IsPrintable(string(r)) {
				return r
			}
			return '_'
		}, text), 0, 64)
		text = strings.Join(strings.FieldsFunc(text, func(r rune) bool {
			return r == '_' || r == ' '
		}), "_")
		return text
	}()

	if filename == "" {
		mname, err := GetMediaFileName(message.Media)
		if err != nil {
			filename = fmt.Sprintf("%d_%s", message.GetID(), xid.New().String())
		} else {
			filename = mname
		}

	}
	return filename + ext
}

func BuildCancelButton(taskID string) tg.KeyboardButtonClass {
	return &tg.KeyboardButtonCallback{
		Text: "取消任务",
		Data: fmt.Appendf(nil, "cancel %s", taskID),
	}
}

func InputMessageClassSliceFromInt(ids []int) []tg.InputMessageClass {
	result := make([]tg.InputMessageClass, 0, len(ids))
	for _, id := range ids {
		result = append(result, &tg.InputMessageID{
			ID: id,
		})
	}
	return result
}

func GetMessagesRange(ctx *ext.Context, chatID int64, minId, maxId int) ([]*tg.Message, error) {
	if minId > maxId {
		return nil, fmt.Errorf("minId (%d) cannot be greater than maxId (%d)", minId, maxId)
	}
	total := maxId - minId + 1
	msgIds := mathutil.Range(minId, total)
	toFetchIds := make([]int, 0, total)
	cached := make(map[int]*tg.Message, total)
	for _, id := range msgIds {
		if msg, ok := cache.Get[*tg.Message](fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, id)); ok {
			cached[id] = msg
		} else {
			toFetchIds = append(toFetchIds, id)
		}
	}
	if len(toFetchIds) == 0 {
		return maputil.Values(cached), nil
	}

	result := make([]*tg.Message, 0, total)
	chunks := slice.Chunk(toFetchIds, 100)
	for _, chunk := range chunks {
		msgs, err := ctx.GetMessages(chatID, InputMessageClassSliceFromInt(chunk))
		if err != nil {
			return nil, err
		}
		if len(msgs) == 0 {
			continue
		}
		for _, msg := range msgs {
			if msg == nil {
				continue
			}
			tgMessage, ok := msg.(*tg.Message)
			if !ok {
				continue
			}
			if tgMessage.GetID() < minId || tgMessage.GetID() > maxId {
				continue
			}
			result = append(result, tgMessage)
		}
	}

	for _, msg := range result {
		cache.Set(fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, msg.GetID()), msg)
	}
	for _, msg := range cached {
		if msg == nil {
			continue
		}
		result = append(result, msg)
	}
	return result, nil
}

type MessageItem struct {
	Message *tg.Message
	Error   error
}

func IterMessages(ctx *ext.Context, chatID int64, minId, maxId int) (<-chan MessageItem, error) {
	total := maxId - minId + 1
	ch := make(chan MessageItem, 100)

	go func() {
		defer close(ch)
		if !ctx.Self.Bot {
			perr := ctx.PeerStorage.GetInputPeerById(chatID)
			if perr == nil || perr.(*tg.InputPeerEmpty) != nil {
				ch <- MessageItem{
					Error: fmt.Errorf("peer not found: %d", chatID),
				}
				return
			}

			for i := 0; i < total; i += 100 {
				start := minId + i
				end := min(start+100, maxId)
				msgs, err := ctx.Raw.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
					Peer:      perr,
					OffsetID:  start,
					AddOffset: start - end,
					Limit:     100,
				})
				if err != nil {
					ch <- MessageItem{
						Error: fmt.Errorf("failed to get messages: %w", err),
					}
					return
				}
				var msgClass []tg.MessageClass
				switch msgsv := msgs.(type) {
				case *tg.MessagesMessages:
					msgClass = msgsv.GetMessages()
				case *tg.MessagesMessagesSlice:
					msgClass = msgsv.GetMessages()
				case *tg.MessagesChannelMessages:
					msgClass = msgsv.GetMessages()
				default:
					ch <- MessageItem{
						Error: fmt.Errorf("unsupported message type: %T", msgsv),
					}
					continue
				}
				for _, msg := range msgClass {
					msg, ok := msg.AsNotEmpty()
					if !ok {
						continue
					}
					switch msg := msg.(type) {
					case *tg.Message:
						key := fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, msg.GetID())
						cache.Set(key, msg)
						ch <- MessageItem{
							Message: msg,
						}
					}
				}
			}
		} else {
			for i := 0; i < total; i += 100 {
				start := minId + i
				end := min(start+100, maxId)
				msgs, err := GetMessagesRange(ctx, chatID, start, end)
				if err != nil {
					ch <- MessageItem{
						Error: fmt.Errorf("failed to get messages: %w", err),
					}
					return
				}
				for _, msg := range msgs {
					if msg == nil {
						continue
					}
					ch <- MessageItem{
						Message: msg,
					}
				}
			}
		}
	}()

	return ch, nil
}

func GetMessageByID(ctx *ext.Context, chatID int64, msgID int) (*tg.Message, error) {
	key := fmt.Sprintf("tgmsg:%d:%d:%d", ctx.Self.ID, chatID, msgID)
	if msg, ok := cache.Get[*tg.Message](key); ok {
		return msg, nil
	}
	msgs, err := ctx.GetMessages(chatID, []tg.InputMessageClass{
		&tg.InputMessageID{ID: msgID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message by ID: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message not found: chatID=%d, msgID=%d", chatID, msgID)
	}
	msg := msgs[0]
	tgm, ok := msg.(*tg.Message)
	if !ok {
		return nil, fmt.Errorf("unexpected message type: %T", msg)
	}
	cache.Set(key, tgm)
	return tgm, nil
}

func GetGroupedMessages(ctx *ext.Context, chatID int64, msg *tg.Message) ([]*tg.Message, error) {
	groupID, isGroup := msg.GetGroupedID()
	if !isGroup || groupID == 0 {
		return nil, fmt.Errorf("message %d is not grouped", msg.GetID())
	}
	msgID := msg.GetID()
	minID := msgID - 10
	maxID := msgID + 10
	if minID < 1 {
		minID = 1
	}
	msgs, err := GetMessagesRange(ctx, chatID, minID, maxID)
	if err != nil {
		return nil, fmt.Errorf("failed to get grouped messages: %w", err)
	}
	groupedMessages := make([]*tg.Message, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		mgid, isGroup := m.GetGroupedID()
		if isGroup && mgid == groupID {
			groupedMessages = append(groupedMessages, m)
		}
	}
	return groupedMessages, nil
}
