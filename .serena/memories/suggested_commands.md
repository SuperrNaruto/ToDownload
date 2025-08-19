# SaveAny-Bot 常用开发命令

## 构建和运行
```bash
# 构建二进制文件
go build -o saveany-bot .

# 直接用Go运行
go run main.go

# 运行前需要配置文件 (需要工作目录中的config.toml)
./saveany-bot

# 使用Docker构建和运行
docker build -t saveany-bot .
docker run -v ./config.toml:/app/config.toml saveany-bot

# 使用docker-compose
docker-compose up
docker-compose -f docker-compose.local.yml up
```

## 测试
```bash
# 运行特定测试
go test ./pkg/queue
go test ./storage/webdav

# 运行所有测试
go test ./...

# 运行测试（详细输出）
go test -v ./...

# 运行测试（包含竞态检测）
go test -race ./...
```

## 依赖管理
```bash
# 安装依赖
go mod download

# 整理依赖
go mod tidy

# 验证依赖
go mod verify

# 查看依赖树
go mod graph
```

## 开发工具
```bash
# 生成国际化文件
go run cmd/geni18n/main.go

# 格式化代码
go fmt ./...

# 代码检查
go vet ./...
```

## 配置和初始化
```bash
# 复制示例配置
cp config.example.toml config.toml

# 查看版本信息
./saveany-bot version
```

## 数据库相关
```bash
# 数据库文件通常位于
./data/session.db  # 会话数据库
```

## 系统命令 (Linux)
```bash
# 查看文件
ls -la
cat filename
head -n 20 filename
tail -f filename

# 搜索
find . -name "*.go"
grep -r "pattern" .
grep -n "pattern" filename

# 进程管理
ps aux | grep saveany-bot
pkill saveany-bot

# 目录操作
cd /path/to/directory
pwd
mkdir dirname
rm -rf dirname

# 文件权限
chmod +x filename
chown user:group filename
```