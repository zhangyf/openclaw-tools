# OpenClaw 腾讯云CVM部署指南

> 本文档记录从零开始在腾讯云CVM上部署OpenClaw的完整过程，包括遇到的问题及解决方案。

---

## 目录

1. [环境准备](#环境准备)
2. [标准化安装步骤](#标准化安装步骤)
3. [问题记录与解决方案](#问题记录与解决方案)
4. [配置优化](#配置优化)
5. [日常维护](#日常维护)

---

## 环境准备

### 服务器规格建议

| 配置项 | 最低配置 | 推荐配置 |
|--------|---------|---------|
| CPU | 1核 | 2核+ |
| 内存 | 2GB | 4GB+ |
| 磁盘 | 20GB SSD | 50GB+ SSD |
| 带宽 | 1Mbps | 5Mbps+ |
| 系统 | Ubuntu 22.04 LTS | Ubuntu 22.04/24.04 LTS |

### 前置检查

```bash
# 确认系统版本
lsb_release -a

# 检查可用内存
free -h

# 检查磁盘空间
df -h

# 检查是否已安装Node.js
node --version  # 需要 v18+
```

---

## 标准化安装步骤

### 步骤1：系统更新与基础工具安装

```bash
# 更新系统包
sudo apt update && sudo apt upgrade -y

# 安装基础工具
sudo apt install -y curl wget git vim htop tmux unzip

# 安装Node.js 22.x（OpenClaw要求v18+）
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs

# 验证Node.js版本
node --version  # 应显示 v22.x.x
npm --version   # 应显示 10.x.x
```

### 步骤2：创建OpenClaw工作目录

```bash
# 创建工作目录
mkdir -p ~/.openclaw/workspace
cd ~/.openclaw/workspace

# 初始化npm项目
npm init -y

# 安装腾讯云COS SDK（用于备份功能）
npm install cos-nodejs-sdk-v5 --save
```

### 步骤3：安装GitHub CLI（可选但推荐）

```bash
# 下载安装gh
mkdir -p ~/.local/bin
cd /tmp
curl -fsSL https://github.com/cli/cli/releases/download/v2.67.0/gh_2.67.0_linux_amd64.tar.gz -o gh.tar.gz
tar -xzf gh.tar.gz
mv gh_2.67.0_linux_amd64/bin/gh ~/.local/bin/
~/.local/bin/gh --version

# 添加到PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### 步骤4：安装OpenClaw

```bash
# 通过npm安装OpenClaw（假设已发布到npm）
sudo npm install -g openclaw

# 或者从源码安装
cd /opt
git clone https://github.com/openclaw/openclaw.git
cd openclaw
npm install
npm run build
npm link
```

### 步骤5：初始化OpenClaw配置

```bash
# 初始化配置
openclaw init

# 编辑配置文件
vim ~/.openclaw/config.json
```

**基础配置示例：**

```json
{
  "gateway": {
    "port": 3000,
    "host": "0.0.0.0"
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "YOUR_BOT_TOKEN"
    }
  },
  "model": {
    "default": "moonshot/kimi-k2.5"
  }
}
```

### 步骤6：配置系统服务（systemd）

```bash
# 创建systemd服务文件
sudo tee /etc/systemd/system/openclaw.service > /dev/null << 'EOF'
[Unit]
Description=OpenClaw Gateway
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/.openclaw/workspace
Environment="HOME=/home/ubuntu"
Environment="OPENCLAW_CONFIG=/home/ubuntu/.openclaw/config.json"
ExecStart=/usr/bin/openclaw gateway start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 重新加载systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start openclaw
sudo systemctl enable openclaw

# 查看状态
sudo systemctl status openclaw
```

### 步骤7：配置防火墙

```bash
# 开放OpenClaw端口（默认3000）
sudo ufw allow 3000/tcp

# 如果使用Telegram Webhook，需要开放HTTPS
sudo ufw allow 443/tcp

# 启用防火墙
sudo ufw enable
```

---

## 问题记录与解决方案

### 问题1：Node.js版本过低

**现象：**
```
error: openclaw@x.x.x: The engine "node" is incompatible with this module.
Expected version ">=18.0.0". Got "12.22.9"
```

**原因：**
Ubuntu默认源的Node.js版本过旧（v12），不满足OpenClaw要求（v18+）。

**解决方案：**
```bash
# 使用NodeSource官方源安装新版Node.js
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs

# 验证版本
node --version  # 应 >= v18
```

---

### 问题2：权限不足导致安装失败

**现象：**
```
npm ERR! Error: EACCES: permission denied, mkdir '/usr/lib/node_modules/openclaw'
```

**原因：**
npm全局安装需要root权限，但直接使用sudo可能导致权限问题。

**解决方案：**

方案A - 使用npx（推荐）：
```bash
# 不全局安装，直接使用npx
npx openclaw gateway start
```

方案B - 修改npm默认目录：
```bash
# 创建本地npm目录
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.profile
source ~/.profile

# 重新安装
npm install -g openclaw
```

---

### 问题3：systemd服务启动失败

**现象：**
```
Failed to start openclaw.service: Unit openclaw.service not found.
```

或

```
openclaw.service: Failed to determine user credentials: No such process
```

**原因：**
1. 服务文件路径或权限问题
2. 用户不存在或配置错误

**解决方案：**
```bash
# 检查服务文件语法
sudo systemd-analyze verify /etc/systemd/system/openclaw.service

# 确保用户存在
id ubuntu  # 或修改为你的实际用户名

# 检查日志
sudo journalctl -u openclaw -f

# 手动测试命令是否能运行
su - ubuntu -c 'openclaw gateway start'
```

---

### 问题4：Telegram Bot连接超时

**现象：**
```
Error: connect ETIMEDOUT 149.154.167.220:443
```

**原因：**
腾讯云CVM在某些地区可能无法直接访问Telegram API服务器。

**解决方案：**

方案A - 配置代理：
```json
// ~/.openclaw/config.json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "YOUR_TOKEN",
      "proxy": {
        "host": "127.0.0.1",
        "port": 7890,
        "protocol": "socks5"
      }
    }
  }
}
```

方案B - 使用Webhook（推荐）：
```bash
# 配置Telegram使用Webhook而不是Long Polling
# 需要配合反向代理（Nginx/Caddy）使用
```

---

### 问题5：COS备份上传失败

**现象：**
```
Error: connect ECONNREFUSED 169.254.0.47:80
```

或

```
Error: InvalidSecretId
```

**原因：**
1. 使用了内网Endpoint但从公网访问
2. 密钥配置错误

**解决方案：**
```bash
# 检查COS配置
# 使用正确的Region和Endpoint

# 正确的配置示例
export TENCENT_COS_SECRET_ID=AKIDxxxxxx
export TENCENT_COS_SECRET_KEY=xxxxxxxx
export TENCENT_COS_REGION=ap-beijing  # 或你的存储桶所在地域

# 验证网络连通性
ping cos.ap-beijing.myqcloud.com
```

---

### 问题6：内存不足导致进程被杀

**现象：**
```
Out of memory: Killed process 12345 (openclaw)
```

**原因：**
CVM内存不足，系统OOM Killer终止了OpenClaw进程。

**解决方案：**

方案A - 增加Swap空间：
```bash
# 创建4GB swap文件
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 持久化
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# 查看swap使用情况
free -h
```

方案B - 限制Node.js内存使用：
```bash
# 在systemd服务中添加内存限制
# /etc/systemd/system/openclaw.service
[Service]
Environment="NODE_OPTIONS=--max-old-space-size=512"
MemoryMax=1G
```

---

### 问题7：GitHub推送被阻止（密钥泄露检测）

**现象：**
```
remote: error: GH013: Repository rule violations found
remote: Push cannot contain secrets
```

**原因：**
GitHub检测到代码中包含腾讯云密钥等敏感信息。

**解决方案：**
```bash
# 1. 立即撤销泄露的密钥（腾讯云控制台）

# 2. 从代码中移除密钥，改为环境变量
# 修改前：
const secretId = 'AKIDxxxxx';

# 修改后：
const secretId = process.env.TENCENT_COS_SECRET_ID;

# 3. 使用.gitignore排除.env文件
echo '.env' >> .gitignore

# 4. 清除Git历史中的密钥
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch scripts/backup-to-cos.js' \
  HEAD

# 或重新初始化git（更彻底）
rm -rf .git
git init
git add .
git commit -m "Initial commit"
git push -f origin main
```

---

## 配置优化

### 1. Nginx反向代理配置

```nginx
# /etc/nginx/sites-available/openclaw
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}
```

### 2. 日志轮转配置

```bash
# /etc/logrotate.d/openclaw
/var/log/openclaw/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0644 ubuntu ubuntu
    sharedscripts
    postrotate
        systemctl reload openclaw
    endscript
}
```

### 3. 自动备份脚本

```bash
#!/bin/bash
# /opt/backup-openclaw.sh

BACKUP_DIR="/backup/openclaw"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份
mkdir -p $BACKUP_DIR
tar -czf $BACKUP_DIR/openclaw_backup_$DATE.tar.gz \
    ~/.openclaw/workspace \
    ~/.openclaw/config.json

# 保留最近30天备份
find $BACKUP_DIR -name "openclaw_backup_*.tar.gz" -mtime +30 -delete

# 可选：上传到COS
# ~/.local/bin/coscli cp $BACKUP_DIR/openclaw_backup_$DATE.tar.gz cos://your-bucket/backups/
```

---

## 日常维护

### 查看服务状态

```bash
# 查看OpenClaw运行状态
sudo systemctl status openclaw

# 查看实时日志
sudo journalctl -u openclaw -f

# 查看最近错误
sudo journalctl -u openclaw --since "1 hour ago" | grep ERROR
```

### 更新OpenClaw

```bash
# 备份当前配置
cp ~/.openclaw/config.json ~/.openclaw/config.json.bak

# 更新到最新版本
npm update -g openclaw

# 重启服务
sudo systemctl restart openclaw
```

### 监控检查清单

- [ ] CPU使用率正常（< 80%）
- [ ] 内存使用率正常（< 80%）
- [ ] 磁盘空间充足（> 20%）
- [ ] Telegram Bot响应正常
- [ ] 备份任务执行成功
- [ ] 日志无异常错误

---

## 附录

### 常用命令速查

```bash
# 启动/停止/重启服务
sudo systemctl start openclaw
sudo systemctl stop openclaw
sudo systemctl restart openclaw

# 查看配置
openclaw config get

# 测试配置
openclaw config validate

# 查看版本
openclaw --version
```

### 相关链接

- OpenClaw官方文档：https://docs.openclaw.ai
- 腾讯云CVM文档：https://cloud.tencent.com/document/product/213
- Node.js版本管理：https://github.com/nvm-sh/nvm

---

**文档版本：** v1.0  
**最后更新：** 2026-02-27  
**适用OpenClaw版本：** v0.x+
