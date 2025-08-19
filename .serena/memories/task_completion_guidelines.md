# SaveAny-Bot 任务完成指南

## 代码更改后的必须步骤

### 1. 代码格式化和检查
```bash
# 格式化所有Go代码
go fmt ./...

# 运行代码检查
go vet ./...

# 整理导入
go mod tidy
```

### 2. 测试验证
```bash
# 运行所有测试
go test ./...

# 运行测试包含竞态检测
go test -race ./...

# 运行特定包的测试
go test ./pkg/queue
go test ./storage/webdav
```

### 3. 构建验证
```bash
# 确保代码能够成功构建
go build -o saveany-bot .

# 验证构建的二进制文件
./saveany-bot --help
```

### 4. 功能验证
```bash
# 如果修改了配置相关代码，验证配置解析
cp config.example.toml config.toml
# 编辑config.toml添加必要的配置（如bot token）
./saveany-bot
```

### 5. Docker构建验证（如果相关）
```bash
# 如果修改了Dockerfile或相关构建逻辑
docker build -t saveany-bot .
```

## 特定类型更改的额外步骤

### 存储后端更改
- 测试特定存储类型的功能
- 验证流式传输模式（如果支持）
- 确保错误处理正确

### 数据库模型更改
- 考虑数据迁移需求
- 验证GORM模型定义
- 测试数据库操作

### 国际化更改
```bash
# 重新生成国际化文件
go run cmd/geni18n/main.go
```

### Bot处理器更改
- 测试Telegram命令功能
- 验证中间件链
- 确保错误处理和恢复机制

## 提交前检查清单
- [ ] 代码格式化完成 (`go fmt`)
- [ ] 代码检查通过 (`go vet`)
- [ ] 所有测试通过 (`go test ./...`)
- [ ] 构建成功 (`go build`)
- [ ] 功能验证完成
- [ ] 相关文档更新（如有需要）
- [ ] 国际化文件更新（如果修改了用户可见文本）

## 部署前验证
- [ ] Docker镜像构建成功
- [ ] 配置文件示例更新（如有新配置项）
- [ ] 性能测试（如果涉及性能相关更改）
- [ ] 向后兼容性检查