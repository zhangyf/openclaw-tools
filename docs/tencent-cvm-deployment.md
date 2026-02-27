# OpenClaw 腾讯云CVM部署指南（实际经验版）

> 本文档基于腾讯云SA5.MEDIUM4实例实际部署经验编写，记录真实遇到的问题及解决方案。

---

## 目录

1. [环境信息](#环境信息)
2. [部署步骤](#部署步骤)
3. [问题记录与解决方案](#问题记录与解决方案)
4. [附录](#附录)

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

### 问题1：systemd user services are unavailable

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
      "allowFrom": [
        USER_ID_NUMBERIC
      ],
      "groupPolicy": "allowlist",
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

### 问题3：群聊中只有我能@机器人，其他人@无响应

**场景：** Telegram群聊中，只有我@机器人时它有响应，群里其他人@它时没有响应

**现象：**
- 我自己@机器人 → 正常响应
- 其他人@机器人 → 无响应
- 私聊机器人 → 正常

**原因：**
配置了 `groupPolicy: "allowlist"` 且 `allowFrom` 只包含了自己的用户ID，导致只有白名单内的用户能触发机器人

**解决方案：**

将 `groupPolicy` 从 `"allowlist"` 改为 `"open"`：

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "YOUR_BOT_TOKEN",
      "groupPolicy": "open",
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
- `groupPolicy: "open"`：对所有人开放，不限制用户
- `groupPolicy: "allowlist"`：仅允许 `allowFrom` 列表中的用户

**重启生效：**
```bash
openclaw gateway restart
```

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

**文档版本：** v1.3（精简版）  
**最后更新：** 2026-02-27  
**适用环境：** 腾讯云SA5.MEDIUM4 | Ubuntu 24.04 LTS
