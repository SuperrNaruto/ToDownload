package tftask

type TaskInfo interface {
	TaskID() string
	FileName() string
	FileSize() int64
	StoragePath() string
	StorageName() string
}

func (t *Task) TaskID() string {
	return t.ID
}

func (t *Task) FileName() string {
	// Use custom name if available, otherwise fall back to original filename
	if t.customName != "" {
		return t.customName
	}
	return t.File.Name()
}

func (t *Task) FileSize() int64 {
	return t.File.Size()
}

func (t *Task) StoragePath() string {
	return t.Path
}

func (t *Task) StorageName() string {
	return t.Storage.Name()
}
