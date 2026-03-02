# OpenClaw 自动化工具集

这是一个为 OpenClaw AI 助手平台开发的自动化工具集合，包含战争简报、任务管理、备份系统等功能。

## 🚀 功能特性

### 1. 战争财经简报系统
- **自动生成**: 每天4个时间点（06:00, 12:00, 18:00, 24:00）自动生成简报
- **详细分析**: 包含战争动态、金融市场反应、投资策略、风险评估
- **多格式输出**: 支持详细Markdown格式和Telegram群聊格式
- **实时搜索**: 集成Tavily搜索API获取最新信息

### 2. 任务管理系统
- **每日任务汇总**: 自动生成任务完成情况报告
- **任务跟踪**: 记录任务进度和检查点
- **优先级管理**: 支持高、中、低优先级任务

### 3. 备份系统
- **自动备份**: 每日23:59自动备份到腾讯云COS
- **Token统计**: 集成AI助手token使用情况统计
- **费用估算**: 基于API使用量估算费用

### 4. 实用工具
- **名称检查**: 自动检查群聊中的称呼规范
- **预聊天检查**: 群聊前的自动化检查
- **Token统计**: 监控AI助手API使用情况

## 📁 项目结构

```
openclaw-tools/
├── scripts/                    # 主要脚本目录
│   ├── war-briefing-detailed.go    # Go语言战争简报脚本（主推）
│   ├── war-briefing-with-finance.js # JavaScript战争简报脚本
│   ├── token-stats.js          # Token使用统计脚本
│   ├── backup-to-cos.js        # 腾讯云COS备份脚本
│   ├── daily-task-report.js    # 每日任务报告脚本
│   ├── task-manager.js         # 任务管理脚本
│   ├── check-yutian-name.js    # 名称检查脚本
│   └── pre-chat-check.js       # 预聊天检查脚本
├── WORKFLOW_AUTO.md           # 自动化工作流程文档
├── README.md                  # 项目说明文档
└── .gitignore                 # Git忽略文件配置
```

## 🛠️ 技术栈

### 主要语言
- **Go**: 用于高性能的战争简报生成（主推）
- **JavaScript/Node.js**: 用于任务管理和备份系统

### 核心特性
- **标准库优先**: 尽量减少外部依赖
- **完整注释**: 所有代码都有详细注释
- **错误处理**: 完善的错误处理和日志记录
- **模块化设计**: 易于维护和扩展

## 📊 战争简报格式

### 详细版简报包含：
1. **战争动态** - 最新战况确认
2. **金融市场反应** - 石油、股市、避险资产实时数据
3. **投资策略建议** - 具体操作建议
4. **风险评估** - 短期、中期风险分析
5. **关键监控点** - 需要关注的重要事件
6. **操作建议总结** - 核心策略总结

### Telegram群聊版：
- 精简格式，适合移动端阅读
- 重点突出，便于快速决策
- 包含紧急程度标识

## 🔧 安装与使用

### 前提条件
1. OpenClaw 平台已安装并配置
2. Tavily API 密钥（用于搜索）
3. 腾讯云COS配置（用于备份）
4. Go 1.20+ 和 Node.js 18+

### 编译Go脚本
```bash
cd scripts
go build -o war-briefing-detailed war-briefing-detailed.go
```

### 配置Cron Job
通过OpenClaw的cron工具配置定时任务：
```bash
openclaw cron add --name "战争简报 12:00" --schedule "0 12 * * *" --command "./scripts/war-briefing-detailed"
```

## ⚙️ 配置说明

### 环境变量
```bash
# Tavily API
export TAVILY_API_KEY=your_tavily_api_key

# 腾讯云COS
export TENCENT_COS_SECRET_ID=your_secret_id
export TENCENT_COS_SECRET_KEY=your_secret_key

# DeepSeek API（用于Token统计）
export DEEPSEEK_API_KEY=your_deepseek_api_key
```

### OpenClaw配置
确保OpenClaw配置文件中包含正确的：
- Telegram bot token
- 群聊ID配置
- 模型API配置

## 📈 性能特点

### Go版本优势
- **执行速度快**: 编译型语言，无需解释器
- **资源占用低**: 内存使用优化
- **部署简单**: 单个可执行文件
- **并发处理**: 支持并发搜索和处理

### 可靠性
- **错误恢复**: 自动重试和降级处理
- **日志记录**: 详细的运行日志
- **状态监控**: 任务执行状态跟踪

## 🤝 贡献指南

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 👥 维护者

- **张老师** - 项目发起者和主要维护者

## 🙏 致谢

- OpenClaw 开发团队
- Tavily 搜索API
- 腾讯云COS服务

---

**最后更新**: 2026年3月2日  
**版本**: 1.0.0