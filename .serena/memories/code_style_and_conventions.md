# SaveAny-Bot 代码风格和约定

## 命名约定
- **包名**: 小写，简短，描述性强 (如 `core`, `storage`, `database`)
- **接口**: 使用 `-able` 或 `-er` 后缀 (如 `Exectable`, `Storage`)
- **函数**: 驼峰命名，公共函数首字母大写
- **变量**: 驼峰命名，私有变量首字母小写
- **常量**: 大写，使用下划线分隔

## 包结构约定
- **cmd/**: 命令行相关代码
- **client/**: 客户端实现（bot, user, middleware）
- **core/**: 核心业务逻辑和任务处理
- **storage/**: 存储后端实现
- **config/**: 配置相关代码
- **database/**: 数据库模型和操作
- **pkg/**: 可重用的包（enums, queue, rule等）
- **common/**: 通用工具和实用程序

## 接口设计模式
```go
// 主要接口都在各自包的主文件中定义
type Storage interface {
    Init(ctx context.Context, cfg storcfg.StorageConfig) error
    Type() storenum.StorageType
    Name() string
    JoinStoragePath(p string) string
    Save(ctx context.Context, reader io.Reader, storagePath string) error
    Exists(ctx context.Context, storagePath string) bool
}

type Exectable interface {
    Type() tasktype.TaskType
    TaskID() string
    Execute(ctx context.Context) error
}
```

## 错误处理
- 使用 `go-faster/errors` 进行错误包装
- 结构化日志使用 `charmbracelet/log`
- 上下文基于取消贯穿整个应用

## 并发模式
- 使用 worker pool 模式处理任务
- 通过 context 进行取消传播
- 使用 sync 包进行并发控制

## 配置模式
- 使用 TOML 格式配置文件
- 通过 Viper 库管理配置
- 支持环境变量覆盖
- 配置验证在启动时进行

## 日志约定
- 使用结构化日志
- 不同级别：Debug, Info, Warn, Error, Fatal
- 包含上下文信息
- 支持中英文日志信息

## 测试约定
- 测试文件命名：`*_test.go`
- 测试包命名：`package_name_test`
- 使用标准 testing 包
- 集成测试使用真实的测试服务器（如WebDAV测试）
- 并发安全测试使用 race detector

## 国际化
- 使用 `go-i18n` 库
- 键定义在 `common/i18n/i18nk/keys.go`
- 翻译文件在 `common/i18n/locale/`
- 支持中文和英文