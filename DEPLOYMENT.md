# 部署指南

## 快速开始

### 1. 克隆仓库
```bash
git clone https://github.com/zhangyf/openclaw-tools.git
cd openclaw-tools
```

### 2. 安装依赖

#### Go 依赖
```bash
cd scripts
go mod download
```

#### 编译 Go 脚本
```bash
go build -o war-briefing-detailed war-briefing-detailed.go
chmod +x war-briefing-detailed
```

#### Node.js 依赖
```bash
npm init -y
# 根据需要安装依赖
```

### 3. 环境配置

创建 `.env` 文件：
```bash
cp .env.example .env
```

编辑 `.env` 文件：
```env
# Tavily API 配置
TAVILY_API_KEY=your_tavily_api_key_here

# 腾讯云 COS 配置
TENCENT_COS_SECRET_ID=your_cos_secret_id
TENCENT_COS_SECRET_KEY=your_cos_secret_key
TENCENT_COS_BUCKET=your_bucket_name
TENCENT_COS_REGION=ap-singapore

# DeepSeek API 配置（用于 Token 统计）
DEEPSEEK_API_KEY=your_deepseek_api_key

# Telegram 配置
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
```

### 4. OpenClaw 集成

#### 配置 Cron Jobs
```bash
# 查看当前 cron jobs
openclaw cron list

# 添加战争简报任务
openclaw cron add \
  --name "战争简报 06:00" \
  --schedule "0 6 * * *" \
  --command "./scripts/war-briefing-detailed" \
  --channel telegram \
  --to "-5149902750"

# 添加每日备份任务
openclaw cron add \
  --name "每日备份 23:59" \
  --schedule "59 23 * * *" \
  --command "node ./scripts/backup-to-cos.js" \
  --channel telegram \
  --to "8331827497"
```

#### 配置 OpenClaw 代理
确保 OpenClaw 配置文件中包含正确的代理配置：
```json
{
  "agents": {
    "list": [
      {
        "id": "mapuedo",
        "workspace": "/path/to/openclaw-tools"
      }
    ]
  }
}
```

## 脚本说明

### 主要脚本

#### 1. `war-briefing-detailed` (Go)
**功能**: 生成详细的战争财经简报
**输出**:
- 详细 Markdown 文件 (`briefings/war-briefing-detailed-YYYY-MM-DD-HH.md`)
- Telegram 格式文件 (`briefings/war-briefing-detailed-YYYY-MM-DD-HH-telegram.md`)
- 控制台输出（适合 cron job 发送）

**使用**:
```bash
./scripts/war-briefing-detailed
```

#### 2. `token-stats.js` (Node.js)
**功能**: 统计 AI 助手 token 使用情况和费用
**输出**:
- JSON 报告 (`token-reports/token-stats-YYYY-MM-DD.json`)
- 文本报告 (`token-reports/token-stats-YYYY-MM-DD.txt`)

**使用**:
```bash
node ./scripts/token-stats.js
```

#### 3. `backup-to-cos.js` (Node.js)
**功能**: 备份到腾讯云 COS，包含 token 统计
**输出**:
- 备份文件到 COS
- 本地备份报告

**使用**:
```bash
node ./scripts/backup-to-cos.js
```

#### 4. `daily-task-report.js` (Node.js)
**功能**: 生成每日任务汇总报告
**使用**:
```bash
node ./scripts/daily-task-report.js
```

### 辅助脚本

#### 1. `check-yutian-name.js`
检查群聊中的称呼规范

#### 2. `pre-chat-check.js`
群聊前的自动化检查

#### 3. `task-manager.js`
任务管理工具

## 定时任务配置

### 推荐的时间表

```bash
# 战争简报（发送到群聊）
0 6 * * *   ./scripts/war-briefing-detailed    # 06:00
0 12 * * *  ./scripts/war-briefing-detailed    # 12:00
0 18 * * *  ./scripts/war-briefing-detailed    # 18:00
0 0 * * *   ./scripts/war-briefing-detailed    # 24:00

# 每日任务报告（发送到私聊）
0 18 * * *   node ./scripts/daily-task-report.js  # 18:00

# 每日备份（包含 token 统计）
59 23 * * *  node ./scripts/backup-to-cos.js      # 23:59
```

### 使用 OpenClaw Cron 系统
```bash
# 查看所有 cron jobs
openclaw cron list

# 添加新 job
openclaw cron add --name "任务名称" --schedule "cron表达式" --command "执行命令"

# 更新现有 job
openclaw cron update --jobId "job_id" --patch '{"name":"新名称","command":"新命令"}'

# 删除 job
openclaw cron remove --jobId "job_id"
```

## 监控和日志

### 日志文件
- 脚本输出日志: `logs/` 目录
- OpenClaw 系统日志: 查看 OpenClaw 日志配置
- Cron job 执行日志: 通过 `openclaw cron runs --jobId <id>` 查看

### 监控建议
1. **定期检查备份状态**
2. **监控 token 使用情况**
3. **检查简报生成质量**
4. **验证任务执行状态**

## 故障排除

### 常见问题

#### 1. Tavily 搜索失败
- 检查 `TAVILY_API_KEY` 环境变量
- 验证网络连接
- 检查 API 配额

#### 2. COS 备份失败
- 检查腾讯云凭证
- 验证 bucket 权限
- 检查网络连接

#### 3. Go 脚本编译失败
- 确保 Go 版本 >= 1.20
- 检查依赖: `go mod tidy`
- 验证文件权限

#### 4. Cron Job 不执行
- 检查 OpenClaw 服务状态: `openclaw gateway status`
- 验证 cron 表达式
- 检查日志: `openclaw cron runs`

### 调试脚本
```bash
# 启用详细日志
DEBUG=* node ./scripts/your-script.js

# 测试 Go 脚本
./scripts/war-briefing-detailed --verbose

# 检查环境变量
env | grep -E "(TAVILY|TENCENT|DEEPSEEK|TELEGRAM)"
```

## 更新和维护

### 更新代码
```bash
git pull origin main
```

### 重新编译 Go 脚本
```bash
cd scripts
go build -o war-briefing-detailed war-briefing-detailed.go
```

### 检查依赖更新
```bash
# Go 依赖
go mod tidy

# Node.js 依赖
npm outdated
npm update
```

## 安全注意事项

1. **保护 API 密钥**: 不要将 `.env` 文件提交到版本控制
2. **权限管理**: 确保脚本有适当的文件权限
3. **日志清理**: 定期清理敏感日志
4. **备份验证**: 定期验证备份文件的完整性

## 支持

如有问题，请：
1. 查看日志文件
2. 检查环境配置
3. 参考 OpenClaw 文档
4. 在 GitHub 仓库提交 Issue

---

**最后更新**: 2026年3月2日  
**版本**: 1.0.0