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

### 问题1：GitHub连接与代码仓库设置

**场景：** 需要将代码推送到GitHub仓库

**遇到的子问题：**
- GitHub CLI安装需要sudo权限，但个人账户没有
- 直接下载二进制文件到 `~/.local/bin` 解决

**解决方案：**
```bash
# 手动安装GitHub CLI（无需sudo）
mkdir -p ~/.local/bin
cd /tmp
curl -fsSL https://github.com/cli/cli/releases/download/v2.67.0/gh_2.67.0_linux_amd64.tar.gz -o gh.tar.gz
tar -xzf gh.tar.gz
mv gh_2.67.0_linux_amd64/bin/gh ~/.local/bin/

# 添加到PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# 验证
git --version
gh --version
```

---

### 问题2：GitHub推送时的密钥泄露检测

**现象：**
```
remote: error: GH013: Repository rule violations found for refs/heads/main.
remote: - GITHUB PUSH PROTECTION
remote: - Push cannot contain secrets
remote: - Tencent Cloud Secret ID detected
```

**原因：**
代码中硬编码了腾讯云COS的SecretId和SecretKey，被GitHub安全扫描拦截。

**解决方案：**

1. **立即修改密钥（腾讯云控制台）**
   - 登录腾讯云控制台 → 访问管理 → API密钥管理
   - 删除或禁用已泄露的密钥
   - 创建新的API密钥

2. **修改代码使用环境变量**

```javascript
// 修改前（错误）
const config = {
    SecretId: 'YOUR_SECRET_ID_HERE',
    SecretKey: 'YOUR_SECRET_KEY_HERE'
};

// 修改后（正确）
const config = {
    SecretId: process.env.TENCENT_COS_SECRET_ID,
    SecretKey: process.env.TENCENT_COS_SECRET_KEY
};

if (!config.SecretId || !config.SecretKey) {
    console.error('错误: 请设置环境变量 TENCENT_COS_SECRET_ID 和 TENCENT_COS_SECRET_KEY');
    process.exit(1);
}
```

3. **创建.env文件并加入.gitignore**

```bash
# 创建.env文件
cat > ~/.openclaw/workspace/.env << 'EOF'
TENCENT_COS_SECRET_ID=你的新SecretId
TENCENT_COS_SECRET_KEY=你的新SecretKey
EOF

# 加入.gitignore
echo '.env' >> ~/.gitignore
```

4. **清除Git历史中的敏感信息**

```bash
cd ~/.openclaw/workspace

# 方法：重新初始化git（最彻底）
rm -rf .git
git init
git config user.name "your-username"
git config user.email "your-email@example.com"
git remote add origin https://github.com/username/repo.git
git add .
git commit -m "Initial commit: clean version without secrets"
git push -f origin main
```

---

### 问题3：定时任务的环境变量加载

**场景：** 配置每日自动备份COS的cron任务

**问题：** cron任务执行时无法读取.env文件中的环境变量

**解决方案：**

**方法1 - 在cron命令中加载环境变量：**

```bash
# 编辑crontab
crontab -e

# 添加任务（加载.env后执行）
59 23 * * * export $(cat /home/username/.openclaw/workspace/.env | xargs) && node /home/username/.openclaw/workspace/scripts/backup-to-cos.js
```

**方法2 - 使用OpenClaw内置cron（推荐）：**

```bash
# OpenClaw的cron任务可以配置前置环境加载
# 参考OpenClaw文档配置定时任务
```

---

### 问题4：个人配置文件的处理

**场景：** 工作目录中有包含个人信息的.md文件（SOUL.md, USER.md, MEMORY.md等）

**问题：** 这些文件包含个人偏好、记忆、身份信息，不应提交到GitHub

**解决方案：**

```bash
# 更新.gitignore
cat >> ~/.openclaw/workspace/.gitignore << 'EOF'

# 个人配置和敏感信息（不提交到GitHub）
AGENTS.md
IDENTITY.md
MEMORY.md
SOUL.md
TOOLS.md
USER.md
HEARTBEAT.md

# 记忆文件（包含个人对话历史）
memory/

# 工作区状态
.openclaw/

# 环境变量
.env
EOF

# 从git历史中移除这些文件
git rm -r --cached AGENTS.md IDENTITY.md MEMORY.md SOUL.md TOOLS.md USER.md HEARTBEAT.md memory/ .openclaw/ 2>/dev/null

# 提交更改
git add .gitignore
git commit -m "security: exclude personal config files from git"
git push origin main
```

---

### 问题5：跨平台/远程操作的便利性

**场景：** 需要在不同场景下与OpenClaw交互

**解决方案：**

1. **Telegram Bot（推荐用于移动端）**
   - 配置Telegram Channel
   - 随时随地通过Telegram与执事对话

2. **保留本地配置文件**
   - 个人配置（USER.md, MEMORY.md等）保留在服务器本地
   - 通过SSH/VS Code Remote编辑

---

### 问题6：Telegram群聊中机器人无法接收@消息

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
