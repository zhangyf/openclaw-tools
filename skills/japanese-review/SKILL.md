---
name: japanese-review
description: 标准日本语初级上册学习复习系统。管理日语词汇数据库(progress.json)、按艾宾浩斯记忆曲线生成每日复习Excel(双版)、录入新词(消息/拍照)、处理错词反馈更新间隔、从课本照片提取知识点并入库。触发场景：(1)老师说"复习"或到每日生成时间；(2)老师说新词或拍照；(3)老师说"错了"或发错词编号；(4)老师说新进度档案需要更新。
---

# 日语复习系统

所有代码和资源均在此 skill 目录下。

## 目录结构

```
skills/japanese-review/
├── SKILL.md                  ← 本说明书
├── scripts/
│   ├── daily-review.go       ← 每日复习生成器源码
│   ├── daily-review          ← 编译好的可执行文件
│   ├── build-vocab.go        ← 初始从Markdown提取数据（已执行）
│   ├── cos_upload.go         ← COS上传工具
│   └── japanese-review.json  ← 配置文件
├── data/
│   ├── progress.json         ← 词汇数据库（源数据）
│   └── review/               ← 生成的复习Excel
└── references/
    ├── 第1课知识点.md ～ 第6课知识点.md
```

## 构建方式（首次）

```bash
cd skills/japanese-review/scripts
go build -o daily-review daily-review.go
```

## 使用方式

```bash
cd skills/japanese-review/scripts
./daily-review                        # 生成今天复习表
./daily-review --config japanese-review.json  # 使用配置（推荐，含COS上传）
./daily-review 07/05                  # 指定日期
```

## 配置

全部参数可配，见 `scripts/japanese-review.json`。配置优先级：
1. `--config 路径.json`
2. 环境变量 `JAPANESE_REVIEW_CONFIG`
3. 内置默认值

## COS

- 桶: `openclaw-backup-tx-1251036673` (北京)
- 端点: `https://openclaw-backup-tx-1251036673.cos.ap-beijing.myqcloud.com`
- 凭据: 环境变量 `TENCENT_COS_SECRET_ID` / `TENCENT_COS_SECRET_KEY`
- 路径: `japanese/progress.json`、`japanese/review/日语复习_YYYYMMDD.xlsx`

## 核心工作流

### 1. 每日复习生成
1. 运行 `./daily-review --config japanese-review.json`
2. 自动计算艾宾浩斯到期词 + 随机抽12句造句
3. 生成双列双版 Excel（默写/核对）
4. 上传到 COS
5. 汇报结果（🔴/🟡/🟢/📝数量）

### 2. 新词录入
- 老师发消息给日语词 → 查中文 → 确认后入库（status=pending）
- 老师拍课本照片 → 识别单词/语法/课文 → 整理知识点笔记 → 确认后入库 → 补造句

### 3. 错词反馈
收到"错了第X号"后：
- 该词 `error_count++`
- `consecutive_correct = 0`
- `last_review = 今天`
- 状态自动变 `red`
- 上传COS同步

### 4. 造句池
预置34句覆盖第1~6课，每天随机抽12句。新增课时需补充对应课造句。

## 艾宾浩斯间隔

| 连续写对次数 | 下次复习间隔 |
|:-----------:|:-----------:|
| 0（刚学或刚错） | 1天 |
| 1次 | 2天 |
| 2次 | 4天 |
| 3次 | 7天 |
| 4次 | 15天 |
| 5次以上 | 30天 |

分类使用原始 status 字段：red=🔴钉子户 / yellow=🟡基本掌握 / green=🟢已掌握
