<div align="center">

# <img src="docs/static/logo.png" width="45" align="center"> Save Any Bot

**简体中文** | [English](https://sabot.unv.app/en/)

一个功能强大的 Telegram 文件保存机器人，支持多种存储后端、AI 智能重命名和动态存储管理。

</div>

## 部署

请参考 [部署文档](https://sabot.unv.app/deployment/installation/)

## ✨ 核心功能

### 📁 文件处理
- **多媒体支持**: 文档、视频、图片、音频、贴纸等全格式支持
- **Telegraph 集成**: 直接保存 Telegraph 页面内容
- **破解限制**: 绕过 Telegram 禁止保存文件的限制
- **批量操作**: 支持批量下载和处理

### 🤖 智能功能
- **AI 智能重命名**: 基于文件内容自动生成有意义的文件名
- **自动分类**: 基于规则的文件自动整理和分类
- **多语言支持**: 国际化界面支持

### 🏗️ 技术特性
- **多用户管理**: 完整的用户权限和隔离系统
- **流式传输**: 支持直接流式传输到存储端，节省本地空间
- **任务队列**: 高效的并发下载和处理系统
- **动态配置**: 运行时动态管理存储配置

### 💾 存储后端
- **Alist**: 强大的网盘系统集成
- **WebDAV**: 标准 WebDAV 协议支持  
- **MinIO/S3**: S3 兼容的对象存储
- **Telegram**: 重传到指定频道或群组
- **本地存储**: 服务器本地文件系统
- **动态管理**: 通过 Bot 界面实时添加、编辑、删除存储配置

## 🚀 快速开始

### Bot 命令
```
/start - 开始使用机器人
/save - 保存当前会话的媒体文件
/dir - 目录管理
/storage_list - 查看存储配置
/storage_add - 添加存储配置
/ai_status - 查看 AI 功能状态
```

### Docker 部署
```bash
# 克隆项目
git clone https://github.com/krau/SaveAny-Bot.git
cd SaveAny-Bot

# 配置文件
cp config.example.toml config.toml
# 编辑 config.toml 添加你的配置

# 启动服务
docker-compose up -d
```

## 🛠️ 技术架构

```
SaveAny-Bot
├── 🤖 Telegram Bot Client    # 用户交互层
├── 📋 Task Queue System      # 任务队列和工作池
├── 🧠 AI Rename Engine       # 智能文件重命名
├── 💾 Storage Abstraction    # 统一存储接口
├── 📊 Database Layer         # SQLite 数据持久化
└── 🔧 Configuration Engine   # 动态配置管理
```

**技术栈**: Go 1.23+ • SQLite • Docker • Telegram Bot API

## 💖 赞助支持

本项目受到 [YxVM](https://yxvm.com/) 与 [NodeSupport](https://github.com/NodeSeekDev/NodeSupport) 的支持.

如果这个项目对你有帮助, 你可以考虑通过以下方式赞助我:

- [爱发电](https://afdian.com/a/unvapp)

## Contributors

<!-- readme: contributors -start -->
<table>
	<tbody>
		<tr>
            <td align="center">
                <a href="https://github.com/krau">
                    <img src="https://avatars.githubusercontent.com/u/71133316?v=4" width="100;" alt="krau"/>
                    <br />
                    <sub><b>Krau</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Silentely">
                    <img src="https://avatars.githubusercontent.com/u/22141172?v=4" width="100;" alt="Silentely"/>
                    <br />
                    <sub><b>Abner</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/TG-Twilight">
                    <img src="https://avatars.githubusercontent.com/u/121682528?v=4" width="100;" alt="TG-Twilight"/>
                    <br />
                    <sub><b>Simon Twilight</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/ysicing">
                    <img src="https://avatars.githubusercontent.com/u/8605565?v=4" width="100;" alt="ysicing"/>
                    <br />
                    <sub><b>缘生</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/AHCorn">
                    <img src="https://avatars.githubusercontent.com/u/42889600?v=4" width="100;" alt="AHCorn"/>
                    <br />
                    <sub><b>安和</b></sub>
                </a>
            </td>
		</tr>
	<tbody>
</table>
<!-- readme: contributors -end -->

## 🙏 致谢

感谢以下开源项目为 SaveAny-Bot 提供的支持:

- **Telegram Libraries**
  - [gotd/td](https://github.com/gotd/td) - 高性能 Telegram Bot API 客户端
  - [celestix/gotgproto](https://github.com/celestix/gotgproto) - Telegram MTProto 协议实现
  - [tdl](https://github.com/iyear/tdl) - Telegram 下载工具参考

- **Storage & Infrastructure**  
  - [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot) - 文件流处理参考
  - [GORM](https://gorm.io) - Go ORM 框架
  - [Cobra](https://github.com/spf13/cobra) - CLI 框架

- **以及所有其他依赖项目的开发者们** 🎉

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE) 开源。

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！**

[📖 完整文档](https://sabot.unv.app) • [🐛 问题反馈](https://github.com/krau/SaveAny-Bot/issues) • [💬 讨论交流](https://github.com/krau/SaveAny-Bot/discussions)

</div>
