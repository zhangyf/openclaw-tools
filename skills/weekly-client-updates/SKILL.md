# 每周客户更新管理 Skill

## 功能描述

本技能用于管理每周客户更新情况，包括：
1. 接收不定期的客户更新信息（一次一个或多个客户）
2. **如实记录老师提供的原文，不做任何润色、改写、精简**
3. 直接操作COS上的文件，不在本地存储
4. 每次更新强制上传到腾讯云COS

## 核心流程

### 1. 接收客户更新
- **输入格式**：用户提供客户更新文本
- **支持模式**：单客户更新或多客户批量更新

### 2. 原文记录（不做润色）
- **严格如实记录**：老师给的文字是什么就记什么
- **禁止做任何润色、改写、精简、扩充**
- 一字不改，原样保存

### 3. 文件管理（纯COS模式）
- **无本地存储**：所有文件读写直接通过COS完成
- **文件命名**：`weekly-updates/YYYY/week-WW.md`
- **COS拉取策略**：
  - 每次先拉取COS上对应周的文件
  - COS上不存在则创建新文件
- **强制上传**：每次更新完成后强制上传到COS

### 4. COS集成
- **桶配置**：通过参数或环境变量指定COS桶信息
- **同步机制**：
  - 每次处理前从COS拉取最新文件
  - 处理完成后强制上传到COS
- **认证方式**：腾讯云SecretId/SecretKey

## 使用方法

### 基本调用
```bash
# 处理单个客户更新
./weekly-client-update --client "客户名" --content "更新内容" --bucket my-bucket

# 处理多个客户更新（JSON格式）
./weekly-client-update --clients '[{"name":"客户A","content":"内容A"}]' --bucket my-bucket
```

### 参数说明
| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `--client` | 是* | 客户名称 | `--client "好未来"` |
| `--content` | 是* | 更新内容 | `--content "本周项目进展..."` |
| `--clients` | 是* | 多个客户JSON | `--clients '[{"name":"A","content":"..."}]'` |
| `--bucket` | 是 | COS桶名称 | `--bucket "my-client-bucket"` |
| `--region` | 否 | COS区域（默认ap-beijing） | `--region "ap-shanghai"` |
| `--week` | 否 | 指定周数（默认当前周） | `--week 18` |
| `--year` | 否 | 指定年份（默认当前年） | `--year 2026` |
| `--secret-id` | 否 | 腾讯云SecretId | `--secret-id "AKID..."` |
| `--secret-key` | 否 | 腾讯云SecretKey | `--secret-key "..."` |

*注：`--client`/`--content` 与 `--clients` 二选一

### 环境变量配置
```bash
# 腾讯云认证（避免与其他技能冲突，使用特定前缀）
export WEEKLY_CLIENT_UPDATE_SECRET_ID="AKID..."
export WEEKLY_CLIENT_UPDATE_SECRET_KEY="..."

# COS默认配置
export WEEKLY_CLIENT_UPDATE_BUCKET="my-client-bucket"
export WEEKLY_CLIENT_UPDATE_REGION="ap-beijing"
export WEEKLY_CLIENT_UPDATE_COS_PATH="weekly-updates/"
```

## 记录规则

### 1. 如实记录
- **一字不改**：老师给的原文，原封不动记录
- **禁止润色**：不做任何改写、精炼、扩展
- **禁止删减**：保留全部原始信息

### 2. 客户识别
- **客户名匹配**：智能识别已存在的客户名
- **更新追加**：新内容追加到已有客户部分
- **重复检测**：避免完全相同的更新内容

## 架构说明

### 纯COS模式
- 不在本地保存任何周报文件
- 处理流程：从COS拉取 → 追加更新 → 强制上传到COS
- 本地仅保留临时文件用于解析

---

**最后更新**：2026-04-30  
**版本**：2.1.0  
**变更**：移除润色功能，改为纯COS模式，无本地存储，每次更新强制上传
