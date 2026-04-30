# 每周客户更新管理工具 v2.1.0

## 变更说明
- **移除所有润色逻辑**：原文照录，一字不改
- **纯COS模式**：不在本地存储周报文件
- **强制上传**：每次更新完成后强制上传到COS
- **COS拉取**：先从COS拉取对应周的文件，不存在则新建

## 快速开始

### 1. 配置环境变量
```bash
export WEEKLY_CLIENT_UPDATE_SECRET_ID="你的SecretId"
export WEEKLY_CLIENT_UPDATE_SECRET_KEY="你的SecretKey"
export WEEKLY_CLIENT_UPDATE_BUCKET="你的存储桶名称"
export WEEKLY_CLIENT_UPDATE_REGION="ap-beijing"
```

### 2. 基本使用

```bash
# 添加单个客户更新
./weekly-client-update \
  --client "客户A" \
  --content "本周项目进展..." \
  --bucket "你的存储桶名称"

# 添加多个客户更新
./weekly-client-update \
  --clients '[{"name":"客户A","content":"内容A"},{"name":"客户B","content":"内容B"}]' \
  --bucket "你的存储桶名称"
```

## COS文件结构
```
weekly-updates/
├── 2026/
│   ├── week-14.md
│   ├── week-15.md
│   ├── week-17.md
│   └── week-18.md
└── ...
```
