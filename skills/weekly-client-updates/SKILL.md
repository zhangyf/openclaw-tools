# 每周客户更新管理 Skill

## 功能描述

本技能用于管理每周客户更新情况，包括：
1. 接收不定期的客户更新信息（一次一个或多个客户）
2. 将原始文字润色到约200字左右，保持自然流畅
3. 合并记录到每周文件中
4. 与腾讯云COS集成，实现文件同步
5. 智能追加更新，避免重复客户条目

## 核心流程

### 1. 接收客户更新
- **输入格式**：用户提供客户更新文本
- **支持模式**：单客户更新或多客户批量更新
- **输入示例**：
  ```
  客户名：好未来
  更新内容：本周启动了内容审核项目，目前供应商五选二，项目金额一年超百万，正在进行第一轮测评...
  ```

### 2. 文本润色
- **润色标准**：约200字，保持自然流畅
- **润色原则**：
  - 不做过多的提炼总结缩减
  - 保持原始信息的完整性和自然度
  - 优化语言表达，使内容更专业流畅
- **格式要求**：自然段落，无需特殊格式化

### 3. 文件管理
- **文件命名**：`客户更新汇总_YYYY年第WW周.md`
  - 示例：`客户更新汇总_2026年第13周.md`
- **存储位置**：
  - 本地：`weekly-updates/`目录
  - 云端：腾讯云COS指定存储桶
- **文件结构**：
  ```markdown
  # 客户更新汇总（2026年第13周）
  
  ## 客户A
  润色后的更新内容...
  
  ## 客户B  
  润色后的更新内容...
  ```

### 4. COS集成
- **桶配置**：通过参数指定COS桶信息
- **同步机制**：
  - 每次处理前从COS拉取最新文件
  - 处理完成后上传更新后的文件
- **认证方式**：腾讯云SecretId/SecretKey

## 使用方法

### 基本调用
```bash
# 处理单个客户更新
/weekly-client-update --client "客户名" --content "更新内容" --bucket "bucket-name"

# 处理多个客户更新（JSON格式）
/weekly-client-update --clients '[{"name":"客户A","content":"内容A"},{"name":"客户B","content":"内容B"}]' --bucket "bucket-name"
```

### 参数说明
| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `--client` | 是* | 客户名称 | `--client "好未来"` |
| `--content` | 是* | 更新内容 | `--content "本周项目进展..."` |
| `--clients` | 是* | 多个客户JSON | `--clients '[{"name":"A","content":"..."}]'` |
| `--bucket` | 是 | COS桶名称 | `--bucket "my-client-bucket"` |
| `--region` | 否 | COS区域（默认ap-beijing） | `--region "ap-shanghai"` |
| `--week` | 否 | 指定周数（默认当前周） | `--week 13` |
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

### 环境变量持久化
环境变量在系统重启后会丢失，建议将配置添加到shell配置文件（如 `~/.bashrc`、`~/.zshrc`）中：

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc
echo 'export WEEKLY_CLIENT_UPDATE_SECRET_ID="AKID..."' >> ~/.bashrc
echo 'export WEEKLY_CLIENT_UPDATE_SECRET_KEY="..."' >> ~/.bashrc
echo 'export WEEKLY_CLIENT_UPDATE_BUCKET="my-client-bucket"' >> ~/.bashrc
echo 'export WEEKLY_CLIENT_UPDATE_REGION="ap-beijing"' >> ~/.bashrc
echo 'export WEEKLY_CLIENT_UPDATE_COS_PATH="weekly-updates/"' >> ~/.bashrc

# 重新加载配置
source ~/.bashrc
```

## 润色算法

### 1. 字数控制
- **目标字数**：约200字（±20%）
- **策略**：
  - 过短内容：适当补充细节，保持自然
  - 过长内容：精简冗余，保留核心信息
  - 恰到好处：仅优化表达，不增减内容

### 2. 语言优化
- **流畅度**：优化句子结构，使表达更自然
- **专业性**：使用适当的商务表达
- **一致性**：保持统一的语言风格

### 3. 客户识别
- **客户名匹配**：智能识别已存在的客户名
- **更新追加**：新内容追加到已有客户部分
- **重复检测**：避免完全相同的更新内容

## 文件操作逻辑

### 1. 文件加载
```go
// 伪代码逻辑
func loadWeeklyFile(year, week int, bucket string) (map[string]string, error) {
    // 1. 检查本地文件是否存在
    // 2. 如果不存在，从COS下载
    // 3. 解析文件内容到客户映射
    // 4. 返回客户名->内容的映射
}
```

### 2. 内容追加
```go
func appendClientUpdate(existingContent, newContent string) string {
    // 1. 如果existingContent为空，直接使用newContent
    // 2. 否则，将newContent追加到existingContent后面
    // 3. 添加适当的分隔符（如空行）
    // 4. 返回合并后的内容
}
```

### 3. 文件保存
```go
func saveWeeklyFile(year, week int, clients map[string]string, bucket string) error {
    // 1. 按模板生成Markdown文件
    // 2. 保存到本地weekly-updates/目录
    // 3. 上传到COS指定桶
    // 4. 返回操作结果
}
```

## 错误处理

### 常见错误
1. **COS连接失败**：检查网络和认证信息
2. **文件不存在**：创建新的周报文件
3. **客户名冲突**：智能合并或提示用户确认
4. **润色失败**：保留原始内容并记录警告

### 恢复机制
- **本地备份**：每次操作前备份当前文件
- **操作日志**：记录详细的操作历史
- **重试机制**：对网络操作实现自动重试

## 示例

### 输入示例
```bash
/weekly-client-update \
  --client "好未来" \
  --content "本周启动了内容审核项目，目前供应商五选二，项目金额一年超百万，正在进行第一轮测评，已进行规则交流，教育场景审核标准为刚需，定制化需求将在讲标阶段展示评分。" \
  --bucket "client-updates-bucket"
```

### 润色后输出（约200字）
```
好未来客户本周正式启动了内容审核项目，目前已进入供应商筛选阶段，共有五家供应商参与竞标，最终将选择两家合作。该项目年度预算超过百万元，显示出客户在内容安全方面的重视程度。当前项目处于第一轮测评阶段，团队已与客户进行了初步的规则交流，明确了教育场景下的审核标准需求。由于教育行业的特殊性，内容审核标准具有刚性需求特点。客户的定制化需求将在后续的讲标阶段进一步展示和评分，以确保最终方案能够精准匹配业务需求。
```

### 文件更新效果
```
# 客户更新汇总（2026年第13周）

## 好未来
好未来客户本周正式启动了内容审核项目，目前已进入供应商筛选阶段...

（后续其他客户更新...）
```

## 技术要求

### 开发语言
- **主语言**：Go（优先使用标准库）
- **依赖**：腾讯云COS SDK、Markdown解析库
- **代码规范**：完整注释，错误处理完善

### 性能要求
- **响应时间**：单次操作<5秒
- **并发支持**：支持多个客户同时更新
- **存储效率**：优化文件读写操作

### 可扩展性
- **插件架构**：支持不同的润色算法
- **存储后端**：可扩展其他云存储服务
- **格式支持**：支持多种输出格式（Markdown、JSON等）

## 维护说明

### 文件结构
```
weekly-client-updates/
├── SKILL.md              # 技能说明文档
├── weekly_client_update.go # 主处理程序
├── cos_client.go         # COS客户端封装
├── text_polisher.go      # 文本润色模块
├── file_manager.go       # 文件管理模块
└── templates/            # 模板文件
    └── weekly_report.tmpl
```

### 测试用例
- 单元测试：各模块功能测试
- 集成测试：COS上传下载测试
- 端到端测试：完整流程测试

---

**最后更新**：2026-04-03  
**版本**：1.0.0  
**开发者**：mapuedo（老师的技术助手）