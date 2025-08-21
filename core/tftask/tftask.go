package tftask

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

type Task struct {
	ID         string
	Ctx        context.Context
	File       tfile.TGFile
	Storage    storage.Storage
	Path       string
	Progress   ProgressTracker
	stream     bool // true if the file should be downloaded in stream mode
	localPath  string
	customName string // custom filename override (e.g., from AI rename)
}

func (t *Task) Type() tasktype.TaskType {
	return tasktype.TaskTypeTgfiles
}

// SetCustomName sets a custom filename override for the task
func (t *Task) SetCustomName(name string) {
	t.customName = name
}

func NewTGFileTask(
	id string,
	ctx context.Context,
	file tfile.TGFile,
	stor storage.Storage,
	path string,
	progress ProgressTracker,
) (*Task, error) {
	_, ok := stor.(storage.StorageCannotStream)
	if !config.Cfg.Stream || ok {
		cachePath, err := filepath.Abs(filepath.Join(config.Cfg.Temp.BasePath, fmt.Sprintf("%s_%s", id, file.Name())))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache: %w", err)
		}
		tftask := &Task{
			ID:        id,
			Ctx:       ctx,
			File:      file,
			Storage:   stor,
			Path:      path,
			Progress:  progress,
			localPath: cachePath,
		}
		return tftask, nil
	}
	tfileTask := &Task{
		ID:       id,
		Ctx:      ctx,
		File:     file,
		Storage:  stor,
		Path:     path,
		Progress: progress,
		stream:   true,
	}
	return tfileTask, nil
}
