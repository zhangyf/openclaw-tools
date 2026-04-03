# 每周客户更新管理工具

用于管理每周客户更新情况，自动润色文本、合并记录并同步到腾讯云COS。

## 功能特点

- **智能润色**: 自动将客户更新润色到约200字，保持自然流畅
- **文件管理**: 每周一个文件，自动合并同一客户的多次更新
- **COS同步**: 自动与腾讯云COS同步，确保数据安全
- **去重机制**: 智能识别并避免重复内容
- **审计追踪**: 保留原始内容记录，便于追溯

## 快速开始

### 1. 编译程序
```bash
chmod +x build.sh
./build.sh
```

### 2. 配置环境变量
```bash
# 腾讯云认证（使用特定前缀避免与其他技能冲突）
export WEEKLY_CLIENT_UPDATE_SECRET_ID="你的SecretId"
export WEEKLY_CLIENT_UPDATE_SECRET_KEY="你的SecretKey"

# COS配置
export WEEKLY_CLIENT_UPDATE_BUCKET="你的存储桶名称"
export WEEKLY_CLIENT_UPDATE_REGION="ap-beijing"  # 默认北京区域
export WEEKLY_CLIENT_UPDATE_COS_PATH="weekly-updates/"  # COS上传路径
```

### 3. 基本使用

#### 添加单个客户更新
```bash
./weekly-client-update \
  --client "好未来" \
  --content "本周启动了内容审核项目，目前供应商五选二，项目金额一年超百万，正在进行第一轮测评..." \
  --bucket "你的存储桶名称"
```

#### 添加多个客户更新（JSON格式）
```bash
./weekly-client-update \
  --clients '[{"name":"客户A","content":"更新内容A"},{"name":"客户B","content":"更新内容B"}]' \
  --bucket "你的存储桶名称"
```

#### 指定特定周次
```bash
./weekly-client-update \
  --client "客户名" \
  --content "更新内容" \
  --bucket "存储桶名称" \
  --year 2026 \
  --week 13
```

## 详细说明

### 命令行参数

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `--client` | 是* | 客户名称 | `--client "好未来"` |
| `--content` | 是* | 更新内容 | `--content "项目进展..."` |
| `--clients` | 是* | 多个客户JSON | `--clients '[{"name":"A","content":"..."}]'` |
| `--bucket` | 是 | COS桶名称 | `--bucket "my-bucket"` |
| `--region` | 否 | COS区域（默认ap-beijing） | `--region "ap-shanghai"` |
| `--secret-id` | 否 | 腾讯云SecretId | `--secret-id "AKID..."` |
| `--secret-key` | 否 | 腾讯云SecretKey | `--secret-key "..."` |
| `--year` | 否 | 年份（默认当前年） | `--year 2026` |
| `--week` | 否 | 周数（默认当前周） | `--week 13` |
| `--help` | 否 | 显示帮助信息 | `--help` |
| `--version` | 否 | 显示版本信息 | `--version` |

*注：`--client`/`--content` 与 `--clients` 二选一

### 环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `WEEKLY_CLIENT_UPDATE_SECRET_ID` | 腾讯云SecretId（避免冲突） | `AKID...` |
| `WEEKLY_CLIENT_UPDATE_SECRET_KEY` | 腾讯云SecretKey（避免冲突） | `...` |
| `WEEKLY_CLIENT_UPDATE_BUCKET` | 默认COS桶名称 | `my-client-bucket` |
| `WEEKLY_CLIENT_UPDATE_REGION` | 默认COS区域 | `ap-beijing` |
| `WEEKLY_CLIENT_UPDATE_COS_PATH` | COS上传路径 | `weekly-updates/` |

### 环境变量持久化
环境变量在系统重启后会丢失，建议将配置添加到shell配置文件中以实现持久化：

#### 1. 添加到 `~/.bashrc` 或 `~/.zshrc`
```bash
# 编辑配置文件
nano ~/.bashrc

# 在文件末尾添加以下配置
export WEEKLY_CLIENT_UPDATE_SECRET_ID="AKID..."
export WEEKLY_CLIENT_UPDATE_SECRET_KEY="..."
export WEEKLY_CLIENT_UPDATE_BUCKET="your-bucket-name"
export WEEKLY_CLIENT_UPDATE_REGION="ap-beijing"
export WEEKLY_CLIENT_UPDATE_COS_PATH="weekly-updates/"

# 使配置生效
source ~/.bashrc
```

#### 2. 使用独立的配置文件（可选）
创建 `~/.weekly-client-update.env` 文件：
```bash
# 创建配置文件
cat > ~/.weekly-client-update.env << EOF
WEEKLY_CLIENT_UPDATE_SECRET_ID="AKID..."
WEEKLY_CLIENT_UPDATE_SECRET_KEY="..."
WEEKLY_CLIENT_UPDATE_BUCKET="your-bucket-name"
WEEKLY_CLIENT_UPDATE_REGION="ap-beijing"
WEEKLY_CLIENT_UPDATE_COS_PATH="weekly-updates/"
EOF

# 每次使用时加载
source ~/.weekly-client-update.env
```

#### 3. 系统服务配置
如果通过systemd服务运行，可在服务文件中配置环境变量。

### 文件结构

#### 本地文件结构
```
weekly-updates/
├── 客户更新汇总_2026年第13周.md
├── 客户更新汇总_2026年第14周.md
└── ...
```

#### COS文件结构
```
weekly-updates/
├── 2026/
│   ├── week-13.md
│   ├── week-14.md
│   └── ...
└── ...
```

#### 周报文件格式
```markdown
# 客户更新汇总（2026年第13周）

> 最后更新: 2026-04-03 15:30:45

## 好未来
好未来客户本周正式启动了内容审核项目...

> **原始记录**: 本周启动了内容审核项目...

## 客户B
客户B的项目进展顺利...

---

**统计**: 本周共更新 2 个客户，总字数约 450 字
```

## 润色算法

### 润色原则
1. **自然流畅**: 优化语言表达，使内容更专业自然
2. **字数控制**: 目标200字（±20%），根据内容调整
3. **信息完整**: 不丢失原始信息，不做过度的提炼总结
4. **商务风格**: 使用适当的商务表达方式

### 润色过程
1. **清理**: 去除多余空格、换行
2. **优化**: 替换非正式表达，优化句子结构
3. **调整**: 根据目标字数扩展或精简内容
4. **整理**: 确保格式规范，以句号结束

## 错误处理

### 常见错误及解决方案

#### 1. COS连接失败
```
错误: 初始化COS客户端失败: 腾讯云认证信息缺失
```
**解决方案**: 检查SecretId和SecretKey配置

#### 2. 文件权限问题
```
错误: 保存文件失败: permission denied
```
**解决方案**: 检查目录写入权限

#### 3. 网络超时
```
警告: COS上传失败（文件已保存到本地）: context deadline exceeded
```
**解决方案**: 文件已保存到本地，可稍后重试上传

### 恢复机制
- **本地备份**: 每次操作前自动备份当前文件
- **断点续传**: 支持从上次中断处继续
- **错误重试**: 对网络操作实现自动重试

## 高级用法

### 批量处理
```bash
# 使用脚本批量处理多个客户
#!/bin/bash
clients=(
  '{"name":"客户A","content":"更新A"}'
  '{"name":"客户B","content":"更新B"}'
)

for client in "${clients[@]}"; do
  ./weekly-client-update --clients "[$client]" --bucket "my-bucket"
done
```

### 集成到工作流
```bash
# 从文件读取客户更新
updates=$(cat updates.json)
./weekly-client-update --clients "$updates" --bucket "my-bucket"
```

### 定时任务
```bash
# 使用cron定时执行
0 18 * * 5 /path/to/weekly-client-update --client "周报汇总" --content "本周客户更新..." --bucket "my-bucket"
```

## 开发说明

### 项目结构
```
weekly-client-updates/
├── main.go              # 主程序入口
├── shared.go            # 共享数据结构
├── file_manager.go      # 文件管理模块
├── cos_client.go        # COS客户端实现
├── go.mod              # 依赖管理
├── go.sum              # 依赖锁文件
├── build.sh            # 编译脚本
└── README.md           # 说明文档
```

### 依赖项
- [腾讯云COS Go SDK](https://github.com/tencentyun/cos-go-sdk-v5)

### 扩展开发

#### 自定义润色器
```go
type CustomPolisher struct{}

func (p *CustomPolisher) Polish(text string, targetLength int) (string, error) {
    // 实现自定义润色逻辑
    return polishedText, nil
}
```

#### 自定义存储后端
```go
type CustomStorage interface {
    Save(report *WeeklyReport) error
    Load(year, week int) (*WeeklyReport, error)
}
```

## 常见问题

### Q: 润色后的内容不符合预期怎么办？
A: 程序会保留原始内容记录，可以在周报文件中查看`> **原始记录**:`部分。

### Q: 如何修改润色算法？
A: 实现`TextPolisher`接口并替换默认的润色器。

### Q: 支持其他云存储吗？
A: 当前只支持腾讯云COS，可通过实现`COSClient`接口扩展其他存储。

### Q: 如何处理历史数据迁移？
A: 可以将历史文件放入`weekly-updates/`目录，程序会自动识别。

## 版本历史

### v1.0.0 (2026-04-03)
- 初始版本发布
- 支持客户更新润色和合并
- 集成腾讯云COS同步
- 提供完整的命令行工具

## 许可证

MIT License

## 支持

如有问题或建议，请提交Issue或联系开发者。