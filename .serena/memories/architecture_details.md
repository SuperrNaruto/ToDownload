# SaveAny-Bot 架构详细信息

## 核心架构组件

### 1. 任务执行系统
- **核心接口**: `Exectable` - 定义可执行任务的契约
- **任务类型**: TfTask（标准文件下载）、BatchTfTask（批量处理）、TphTask（Telegraph内容处理）
- **任务队列**: 优先级队列，支持取消和并发处理
- **工作池**: 可配置的goroutine池，并发处理任务

### 2. 存储抽象层
```go
type Storage interface {
    Init(ctx context.Context, cfg storcfg.StorageConfig) error
    Type() storenum.StorageType
    Name() string
    JoinStoragePath(p string) string
    Save(ctx context.Context, reader io.Reader, storagePath string) error
    Exists(ctx context.Context, storagePath string) bool
}
```

**支持的存储类型**:
- `local/`: 本地文件系统存储
- `alist/`: Alist API集成
- `webdav/`: WebDAV协议支持
- `minio/`: S3兼容对象存储
- `telegram/`: Telegram文件上传后端

### 3. 规则系统
```go
type RuleClass[InputType any] interface {
    Type() ruleenum.RuleType
    Match(input InputType) (bool, error)
    StorageName() string
    StoragePath() string
}
```

### 4. 配置系统
- **格式**: TOML配置文件
- **主要部分**:
  - `[telegram]`: Bot令牌、API凭证、代理设置
  - `[[storages]]`: 存储端点配置数组
  - `[[users]]`: 用户访问控制定义数组
  - 全局设置: workers、重试限制、流模式

### 5. 数据库层
- **ORM**: GORM with SQLite
- **主要模型** (database/model.go):
  - 用户管理与存储权限
  - 聊天跟踪和目录管理
  - 基于规则的文件组织
  - 任务进度跟踪

## 关键设计模式

### 工厂模式
```go
var storageConstructors = map[storenum.StorageType]StorageConstructor{
    storenum.Alist:    func() Storage { return new(alist.Alist) },
    storenum.Local:    func() Storage { return new(local.Local) },
    // ...
}
```

### 中间件链模式
Bot处理器使用中间件链进行请求处理：
- 认证中间件
- 洪水控制中间件
- 恢复中间件
- 重试中间件

### 观察者模式
任务执行带有钩子系统：
- 任务开始前钩子
- 任务成功钩子
- 任务失败钩子
- 任务取消钩子

## 流式传输模式
当启用时，文件直接流向存储而不进行本地缓存：
- 减少磁盘使用但增加失败率
- 不是所有存储后端都支持
- 禁用多线程下载

## 并发模型
- **Worker Pool**: 可配置的goroutine数量处理下载任务
- **Context传播**: 整个应用程序中基于上下文的取消
- **任务队列**: 线程安全的优先级队列，支持并发操作

## 错误处理策略
- 使用 `go-faster/errors` 进行错误包装
- 结构化日志 `charmbracelet/log`
- 重试机制与指数退避
- 优雅降级策略

## 国际化架构
- 使用 `go-i18n` 进行多语言支持
- 键定义在 `common/i18n/i18nk/keys.go`
- 翻译文件在 `common/i18n/locale/`
- 运行时语言切换支持