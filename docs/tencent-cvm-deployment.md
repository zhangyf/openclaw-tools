# OpenClaw 腾讯云CVM部署指南（实际经验版）

> 本文档基于腾讯云SA5.MEDIUM4实例实际部署经验编写，记录真实遇到的问题及解决方案。

---

## 目录

1. [环境信息](#环境信息)
2. [部署步骤](#部署步骤)
3. [问题记录与解决方案](#问题记录与解决方案)
4. [配置优化](#配置优化)
5. [日常维护](#日常维护)

---

## 环境信息

### 服务器规格

| 配置项 | 实际配置 |
|--------|---------|
| 实例类型 | 腾讯云标准型SA5 |
| 规格 | SA5.MEDIUM4 |
| CPU | 2核 |
| 内存 | 4GB |
| 磁盘 | 50GB SSD |
| 系统 | Ubuntu Server 24.04 LTS 64位 |
| 地域 | 新加坡 |

### 前置准备

- 已创建非root个人账户
- 个人账户已配置sudo权限
- 安全组已开放必要端口（按需）

---

## 部署步骤

### 步骤1：系统更新

```bash
# 更新系统软件包
sudo apt-get update
sudo apt-get upgrade -y
```

### 步骤2：官方一键安装

```bash
# 执行OpenClaw官方安装脚本
curl -fsSL https://openclaw.ai/install.sh | bash
```

安装完成后，OpenClaw会自动：
- 安装必要的依赖
- 创建工作目录 `~/.openclaw/workspace`
- 初始化基础配置

### 步骤3：验证安装

```bash
# 检查版本
openclaw --version

# 检查服务状态
openclaw gateway status
```

### 步骤4：配置（按需）

```bash
# 编辑配置文件
vim ~/.openclaw/config.json
```

---

## 问题记录与解决方案

### 问题1：systemd user services 不可用

**场景：** 执行 `openclaw gateway install` 安装系统服务时失败

**现象：**
```
Error: systemd user services are unavailable
# 或
Failed to connect to bus: No such file or directory
```

**原因：**
- 用户未启用 linger 模式（允许用户退出登录后继续运行服务）
- XDG_RUNTIME_DIR 环境变量未正确设置

**解决方案：**

按顺序执行以下命令：

```bash
# 1. 启用用户的 linger 模式
sudo loginctl enable-linger $(whoami)

# 2. 设置运行时目录环境变量
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 3. 重新安装 OpenClaw 服务
openclaw gateway install --force

# 4. 启动服务
openclaw gateway start
```

**持久化配置：**

将环境变量添加到 `~/.bashrc` 避免每次手动设置：

```bash
echo 'export XDG_RUNTIME_DIR=/run/user/$(id -u)' >> ~/.bashrc
source ~/.bashrc
```

**验证修复：**

```bash
# 检查服务状态
systemctl --user status openclaw

# 查看日志
journalctl --user -u openclaw -f
```

---

### 问题2：Telegram群聊中机器人无法接收@消息

**场景：** OpenClaw机器人加入Telegram群聊后，群里@机器人的消息收不到

**现象：**
- 私聊机器人正常
- 群聊中@机器人无响应
- 日志中看不到群消息

**原因：**
OpenClaw默认配置下，群聊消息处理策略需要显式配置

**解决方案：**

在 `~/.openclaw/config.json` 中添加群聊配置：

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "YOUR_BOT_TOKEN",
      "groups": {
        "*": {
          "requireMention": true
        }
      }
    }
  }
}
```

**配置说明：**
- `groups`: 群聊配置对象
- `"*"`: 通配符，表示应用于所有群聊
- `requireMention: true`: 需要被@提及才响应

**重启生效：**
```bash
openclaw gateway restart
```

**参考配置来源：** 腾讯云开发者社区文章 - https://cloud.tencent.com/developer/article/2626214

**场景：** OpenClaw机器人加入Telegram群聊后，群里@机器人的消息收不到

**现象：**
- 私聊机器人正常
- 群聊中@机器人无响应
- 日志中看不到群消息

**原因：**
OpenClaw默认配置下，群聊消息处理策略需要显式配置

**解决方案：**

在 `~/.openclaw/config.json` 中添加群聊配置：

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "YOUR_BOT_TOKEN",
      "groups": {
        "*": {
          "requireMention": true
        }
      }
    }
  }
}
```

**配置说明：**
- `groups`: 群聊配置对象
- `"*"`: 通配符，表示应用于所有群聊
- `requireMention: true`: 需要被@提及才响应

**重启生效：**
```bash
openclaw gateway restart
```

**参考配置来源：** 腾讯云开发者社区文章 - https://cloud.tencent.com/developer/article/2626214

---

## 配置优化

### 1. 建议的目录结构

```
~/.openclaw/workspace/
├── docs/                    # 文档（提交到GitHub）
│   └── tencent-cvm-deployment.md
├── scripts/                 # 工具脚本（提交到GitHub）
│   └── backup-to-cos.js
├── .github/                 # GitHub配置（可选）
├── .gitignore              # Git忽略规则
├── package.json            # 依赖管理
└── README.md               # 项目说明

~/.openclaw/
├── config.json             # OpenClaw配置
├── .env                    # 环境变量（不提交）
├── AGENTS.md               # 个人配置（不提交）
├── IDENTITY.md             # 身份信息（不提交）
├── MEMORY.md               # 长期记忆（不提交）
├── SOUL.md                 # 人设定义（不提交）
├── TOOLS.md                # 工具备注（不提交）
├── USER.md                 # 用户信息（不提交）
├── HEARTBEAT.md            # 心跳任务（不提交）
└── memory/                 # 每日记忆（不提交）
    └── 2026-02-27.md
```

### 2. 备份脚本（优化版）

```javascript
#!/usr/bin/env node
// scripts/backup-to-cos.js

/**
 * OpenClaw Daily Backup Script
 * 备份配置和会话记忆到腾讯云COS
 * 
 * 环境变量配置:
 * - TENCENT_COS_SECRET_ID: 腾讯云SecretId
 * - TENCENT_COS_SECRET_KEY: 腾讯云SecretKey
 * - TENCENT_COS_BUCKET: 存储桶名称（可选，默认openclaw-bakup-1251036673）
 * - TENCENT_COS_REGION: 地域（可选，默认ap-singapore）
 */

const COS = require('cos-nodejs-sdk-v5');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// COS配置 - 从环境变量读取
const config = {
    Bucket: process.env.TENCENT_COS_BUCKET || 'openclaw-bakup-1251036673',
    Region: process.env.TENCENT_COS_REGION || 'ap-singapore',
    SecretId: process.env.TENCENT_COS_SECRET_ID,
    SecretKey: process.env.TENCENT_COS_SECRET_KEY
};

if (!config.SecretId || !config.SecretKey) {
    console.error('[ERROR] 请设置环境变量 TENCENT_COS_SECRET_ID 和 TENCENT_COS_SECRET_KEY');
    process.exit(1);
}

const cos = new COS({
    SecretId: config.SecretId,
    SecretKey: config.SecretKey
});

const WORKSPACE_DIR = path.join(process.env.HOME, '.openclaw/workspace');
const BACKUP_DIR = '/tmp/openclaw-backup';

async function backup() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const dateStr = new Date().toISOString().split('T')[0];
    
    console.log(`[${new Date().toISOString()}] 开始备份...`);
    
    // 创建临时备份目录
    if (!fs.existsSync(BACKUP_DIR)) {
        fs.mkdirSync(BACKUP_DIR, { recursive: true });
    }
    
    const tarFileName = `openclaw-backup-${dateStr}-${timestamp}.tar.gz`;
    const tarPath = path.join(BACKUP_DIR, tarFileName);
    
    try {
        // 打包工作目录
        console.log(`[${new Date().toISOString()}] 打包文件...`);
        execSync(
            `tar -czf "${tarPath}" -C "${path.dirname(WORKSPACE_DIR)}" --exclude='node_modules' --exclude='.git' "${path.basename(WORKSPACE_DIR)}"`,
            { stdio: 'inherit' }
        );
        
        console.log(`[${new Date().toISOString()}] 打包完成: ${tarPath}`);
        
        // 上传到COS
        console.log(`[${new Date().toISOString()}] 上传到COS...`);
        
        const fileContent = fs.readFileSync(tarPath);
        const cosKey = `backups/${dateStr}/${tarFileName}`;
        
        await new Promise((resolve, reject) => {
            cos.putObject({
                Bucket: config.Bucket,
                Region: config.Region,
                Key: cosKey,
                Body: fileContent,
                ContentLength: fileContent.length
            }, (err, data) => {
                if (err) {
                    reject(err);
                } else {
                    resolve(data);
                }
            });
        });
        
        console.log(`[${new Date().toISOString()}] 上传成功: cos://${config.Bucket}/${cosKey}`);
        
        // 清理临时文件
        fs.unlinkSync(tarPath);
        console.log(`[${new Date().toISOString()}] 备份完成！`);
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 备份失败:`, error.message);
        process.exit(1);
    }
}

backup();
```

### 3. 定时任务配置

```bash
# 使用OpenClaw内置cron配置每日23:59备份
# 配置示例（在OpenClaw配置中）：
{
  "cron": {
    "jobs": [
      {
        "name": "daily-backup",
        "schedule": "59 23 * * *",
        "command": "export $(cat /home/username/.openclaw/workspace/.env | xargs) && node /home/username/.openclaw/workspace/scripts/backup-to-cos.js"
      }
    ]
  }
}
```

---

## 日常维护

### 常用命令

```bash
# 查看OpenClaw状态
openclaw status

# 查看日志
openclaw logs -f

# 重启服务
openclaw gateway restart

# 更新OpenClaw
openclaw update
```

### 备份检查

```bash
# 手动测试备份
export $(cat ~/.openclaw/workspace/.env | xargs) && node ~/.openclaw/workspace/scripts/backup-to-cos.js

# 检查COS中的备份列表
# 登录腾讯云控制台或使用coscli工具
```

---

## 部署总结

### 实际部署流程回顾

1. ✅ 创建SA5.MEDIUM4实例（Ubuntu 24.04 LTS）
2. ✅ 创建个人账户并配置sudo权限
3. ✅ 系统更新（apt update && apt upgrade）
4. ✅ 执行官方安装脚本（curl | bash）
5. ✅ 配置GitHub CLI（手动安装到~/.local/bin）
6. ✅ 创建openclaw-tools仓库
7. ✅ 处理密钥泄露问题（改为环境变量）
8. ✅ 配置.gitignore排除个人文件
9. ✅ 配置每日自动备份到COS

### 关键经验

1. **不要硬编码密钥** - 始终使用环境变量
2. **及时撤销泄露密钥** - 一旦提交到GitHub立即在云平台撤销
3. **个人账户+sudo** - 比直接用root更安全
4. **.gitignore要谨慎** - 个人配置文件不要提交

---

## 附录

### Telegram Bot 配置参考

> OpenClaw 连接 Telegram Bot 的详细配置，可参考腾讯云开发者社区文章：
> **《玩转OpenClaw｜云上OpenClaw(Clawdbot)快速接入Telegram指南》**
> https://cloud.tencent.com/developer/article/2626214

该文章涵盖：
- Telegram Bot 创建与配置
- Webhook 模式设置
- 腾讯云轻量服务器部署
- 常见问题排查

### 相关链接

- OpenClaw 官方文档：https://docs.openclaw.ai
- OpenClaw GitHub：https://github.com/openclaw/openclaw
- 腾讯云 CVM 文档：https://cloud.tencent.com/document/product/213
- 腾讯云开发者社区 OpenClaw 专栏：https://cloud.tencent.com/developer/column/96998

---

**文档版本：** v1.2（实际经验版）  
**最后更新：** 2026-02-27  
**适用环境：** 腾讯云SA5.MEDIUM4 | Ubuntu 24.04 LTS
