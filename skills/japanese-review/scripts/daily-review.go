// daily-review.go — 根据艾宾浩斯记忆曲线生成每日日语复习 Excel
// 输出双版 .xlsx: Sheet1(仅中文→默写) / Sheet2(中文+日语→核对)
//
// 使用方式:
//   ./daily-review                          // 生成今天复习表
//   ./daily-review 07/05                    // 指定日期
//   ./daily-review --config 自定义路径.json  // 指定配置
//
// 所有参数有默认值且可通过配置文件覆盖。
// 配置文件路径优先级: --config > JAPANESE_REVIEW_CONFIG env > 内置默认值
// 示例配置见 japanese-review.json

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/xuri/excelize/v2"
)

// ============================================================
// 配置（默认值 + 可覆盖）
// ============================================================

// Config 程序配置，所有字段均可通过 JSON 文件覆盖
type Config struct {
	// 本地路径
	ProgressFile string `json:"progress_file"` // 词汇数据库路径
	OutputDir    string `json:"output_dir"`    // 复习Excel输出目录
	NotesDir     string `json:"notes_dir"`     // 知识点笔记目录

	// 云端(可选)
	COS struct {
		Enabled    bool   `json:"enabled"`
		BucketURL  string `json:"bucket_url"`
		Region     string `json:"region"`
		Progress   string `json:"progress_path"`
		ReviewFmt  string `json:"review_fmt"` // 格式化串，%s 会被替换为 YYYYMMDD
	} `json:"cos"`

	// 艾宾浩斯间隔表 key=连续对次数, value=间隔天数
	Ebbinghaus map[int]int `json:"ebbinghaus"`

	// 绿色单词抽样
	Green struct {
		MaxBeforeSample int `json:"max_before_sample"` // 超过此数开始抽样
		SampleFraction  int `json:"sample_fraction"`   // 取 1/N
		MinSample       int `json:"min_sample"`        // 最少保留
	} `json:"green"`

	// 造句
	Sentences struct {
		DailyCount int `json:"daily_count"` // 每天随机出几道
	} `json:"sentences"`
}

// defaultConfig 返回内置默认配置
func defaultConfig() Config {
	var cfg Config
	cfg.ProgressFile = "/home/zhangyufeng/.openclaw/workspace/skills/japanese-review/data/progress.json"
	cfg.OutputDir = "/home/zhangyufeng/.openclaw/workspace/skills/japanese-review/data/review"
	cfg.NotesDir = "/home/zhangyufeng/.openclaw/workspace/skills/japanese-review/references"

	cfg.COS.BucketURL = "https://openclaw-backup-tx-1251036673.cos.ap-beijing.myqcloud.com"
	cfg.COS.Region = "ap-beijing"
	cfg.COS.Progress = "japanese/progress.json"
	cfg.COS.ReviewFmt = "japanese/review/日语复习_%s.xlsx"

	cfg.Ebbinghaus = map[int]int{
		0: 1, // 刚错或刚学
		1: 2,
		2: 4,
		3: 7,
		4: 15,
		5: 30,
	}
	// 5次以上也用30天
	for i := 6; i <= 100; i++ {
		cfg.Ebbinghaus[i] = 30
	}

	cfg.Green.MaxBeforeSample = 10
	cfg.Green.SampleFraction = 3 // 取 1/3
	cfg.Green.MinSample = 5

	cfg.Sentences.DailyCount = 12

	return cfg
}

// loadConfig 加载配置：默认值 + 可选文件覆盖
func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("读取配置失败: %w", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("解析配置失败: %w", err)
		}
	}
	return cfg, nil
}

// ============================================================
// 数据结构
// ============================================================

// Word 单词学习数据
type Word struct {
	Japanese           string `json:"japanese"`
	Chinese            string `json:"chinese"`
	ReviewCount        int    `json:"review_count"`
	ErrorCount         int    `json:"error_count"`
	ConsecutiveCorrect int    `json:"consecutive_correct"`
	LastReview         string `json:"last_review"`
	Status             string `json:"status"` // green / yellow / red / pending
}

// Sentence 造句
type Sentence struct {
	ID                 string `json:"id"`
	Chinese            string `json:"chinese"`
	Japanese           string `json:"japanese"`
	ReviewCount        int    `json:"review_count"`
	ErrorCount         int    `json:"error_count"`
	ConsecutiveCorrect int    `json:"consecutive_correct"`
	LastReview         string `json:"last_review"`
	Status             string `json:"status"`
}

// VocabDB 词汇数据库（本机持久化 + COS 同步）
type VocabDB struct {
	Words     []Word     `json:"words"`
	Sentences []Sentence `json:"sentences"`
	Updated   string     `json:"updated"`
}

// DueItem 到期的复习项
type DueItem struct {
	Seq      int    // 序号
	Chinese  string
	Japanese string
	Category string // red / yellow / green / sentence
}

// ============================================================
// 日期工具
// ============================================================

// parseDate 解析多种日期格式
func parseDate(s string) (time.Time, error) {
	formats := []string{"2006-01-02", "01/02", "1/2", "2006/01/02"}
	now := time.Now()
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err != nil {
			continue
		}
		if f == "01/02" || f == "1/2" {
			t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("无法解析日期: %s", s)
}

// parseMMDD 解析 MM/DD 格式日期，自动推算年份
func parseMMDD(s string, ref time.Time) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("空日期")
	}
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("格式错误: %s", s)
	}
	var m, d int
	if _, err := fmt.Sscanf(parts[0], "%d", &m); err != nil {
		return time.Time{}, fmt.Errorf("月份错误: %s", s)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &d); err != nil {
		return time.Time{}, fmt.Errorf("日期错误: %s", s)
	}

	year := ref.Year()
	// 跨年修正：参考日期在1月但上次复习在12月 → 去年
	if ref.Month() < time.July && time.Month(m) > time.June {
		year--
	}
	return time.Date(year, time.Month(m), d, 0, 0, 0, 0, ref.Location()), nil
}

// ============================================================
// 艾宾浩斯间隔（配置驱动）
// ============================================================

func getInterval(cfg Config, consecutive int) int {
	// 查找精确匹配
	if days, ok := cfg.Ebbinghaus[consecutive]; ok {
		return days
	}
	// 未找到则取最大值（超过配置表上限）
	maxDays := 30
	for _, v := range cfg.Ebbinghaus {
		if v > maxDays {
			maxDays = v
		}
	}
	return maxDays
}

// ============================================================
// 到期判定
// ============================================================

// isDue 判断单词今天是否到期
func isDue(cfg Config, w Word, today time.Time) bool {
	if w.LastReview == "" {
		return true // 从未复习，到期
	}
	lastRev, err := parseMMDD(w.LastReview, today)
	if err != nil {
		return true
	}
	interval := getInterval(cfg, w.ConsecutiveCorrect)
	dueDate := lastRev.AddDate(0, 0, interval)

	dueStart := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, today.Location())
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	return !dueStart.After(todayStart)
}

// computeCategory 确定复习分类
// 直接使用进度档案中的原始状态字段，与IMA保持一致
func computeCategory(w Word) string {
	switch w.Status {
	case "green":
		return "green"
	case "yellow", "pending":
		return "yellow"
	case "red":
		return "red"
	default:
		return "yellow"
	}
}

// ============================================================
// Excel 生成
// ============================================================

func generateReviewExcel(cfg Config, items []DueItem, sentences []DueItem, outputPath string) error {
	f := excelize.NewFile()

	// ---- 样式工厂 ----
	style := func(s *excelize.Style) int {
		id, err := f.NewStyle(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  样式创建失败: %v\n", err)
		}
		return id
	}

	headerS := style(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "000000"},
			{Type: "right", Style: 1, Color: "000000"},
			{Type: "top", Style: 1, Color: "000000"},
			{Type: "bottom", Style: 1, Color: "000000"},
		},
	})
	sectionYellowS := style(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
	})
	sectionRedS := style(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#C00000"}, Pattern: 1},
	})
	sectionGreenS := style(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#548235"}, Pattern: 1},
	})
	sectionSentenceS := style(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#7030A0"}, Pattern: 1},
	})
	dataS := style(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "000000"},
			{Type: "right", Style: 1, Color: "000000"},
			{Type: "top", Style: 1, Color: "000000"},
			{Type: "bottom", Style: 1, Color: "000000"},
		},
	})

	sectionStyle := func(title string) int {
		switch {
		case strings.Contains(title, "🔴"):
			return sectionRedS
		case strings.Contains(title, "🟢"):
			return sectionGreenS
		case strings.Contains(title, "📝"):
			return sectionSentenceS
		default:
			return sectionYellowS
		}
	}

	type section struct {
		title string
		items []DueItem
	}

	// 双列写入通用函数（仿IMA格式：一行两个词）
	// 布局：A=序号1 B=中文1 C=日语1 D=序号2 E=中文2 F=日语2
	writeRows := func(sheetName string, sections []section, fillJapanese bool) int {
		f.SetSheetName("Sheet1", sheetName)
		row := 1
		for _, sec := range sections {
			if len(sec.items) == 0 {
				continue
			}
			// 分区标题（跨A-F合并）
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("%s %d词", sec.title, len(sec.items)))
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle(sec.title))
			row++

			// 表头：双列
			headers := []string{"序号", "中文", "日语", "序号", "中文", "日语"}
			for ci, h := range headers {
				cell := fmt.Sprintf("%c%d", 'A'+ci, row)
				f.SetCellValue(sheetName, cell, h)
				f.SetCellStyle(sheetName, cell, cell, headerS)
			}
			row++

			// 数据：仿IMA布局——左列=上半段，右列=下半段
			// 序号重新按列编号：左列1~half，右列half+1~N
			total := len(sec.items)
			half := (total + 1) / 2 // 向上取整让左列多一个
			for i := 0; i < half; i++ {
				leftIdx := i
				rightIdx := i + half

				// 左侧词（列内序号 = i+1）
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), sec.items[leftIdx].Chinese)
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), dataS)
				if fillJapanese {
					f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), sec.items[leftIdx].Japanese)
				}

				// 右侧词（列内序号 = half + i + 1）
				if rightIdx < total {
					f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), half+i+1)
					f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), sec.items[rightIdx].Chinese)
					f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("F%d", row), dataS)
					if fillJapanese {
						f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), sec.items[rightIdx].Japanese)
					}
				}
				row++
			}
			row++ // 空行
		}
		return row
	}

	// 按分类整理
	sections := []section{
		{"🔴 钉子户（今日到期）", filterBy(items, "red")},
		{"🟡 今日到期", filterBy(items, "yellow")},
		{"🟢 到期抽查", filterBy(items, "green")},
	}

	// ======== Sheet1: 默写 ========
	writeRows("默写", sections, false)

	// ======== Sheet2: 核对 ========
	f.NewSheet("核对")
	writeRows("核对", sections, true)

	// ======== 造句（追加在两个Sheet末尾） ========
	if len(sentences) > 0 {
		for _, sheetName := range []string{"默写", "核对"} {
			rows, _ := f.GetRows(sheetName)
			row := len(rows) + 2

			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("📝 造句 共%d句", len(sentences)))
			f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionSentenceS)
			row++

			for ci, h := range []string{"序号", "中文提示", "日语"} {
				f.SetCellValue(sheetName, fmt.Sprintf("%c%d", 'A'+ci, row), h)
				f.SetCellStyle(sheetName, fmt.Sprintf("%c%d", 'A'+ci, row), fmt.Sprintf("%c%d", 'A'+ci, row), headerS)
			}
			row++

			isCheck := sheetName == "核对"
			for _, s := range sentences {
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("S%d", s.Seq))
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), s.Chinese)
				if isCheck {
					f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), s.Japanese)
				}
				f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), dataS)
				row++
			}
		}
	}

	// 列宽（双列布局：A-C + D-F）
	for _, sheet := range []string{"默写", "核对"} {
		f.SetColWidth(sheet, "A", "A", 6)
		f.SetColWidth(sheet, "B", "B", 22)
		f.SetColWidth(sheet, "C", "C", 24)
		f.SetColWidth(sheet, "D", "D", 6)
		f.SetColWidth(sheet, "E", "E", 22)
		f.SetColWidth(sheet, "F", "F", 24)
	}

	return f.SaveAs(outputPath)
}

func filterBy(items []DueItem, cat string) []DueItem {
	var out []DueItem
	for _, it := range items {
		if it.Category == cat {
			out = append(out, it)
		}
	}
	return out
}

// ============================================================
// COS 上传
// ============================================================

func tryUploadToCOS(cfg Config, localPath, cosPath string) {
	if !cfg.COS.Enabled {
		return
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  读取 %s 失败: %v\n", localPath, err)
		return
	}
	client, err := newCosClient(cfg.COS.BucketURL, cfg.COS.Region)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  COS 客户端创建失败: %v\n", err)
		return
	}
	if err := cosPutObject(client, cosPath, data); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  COS 上传失败 [%s]: %v\n", cosPath, err)
		return
	}
	fmt.Printf("  ☁️  已上传COS: %s\n", cosPath)
}

func newCosClient(bucketURL, _ string) (*cos.Client, error) {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, fmt.Errorf("解析 BucketURL 失败: %w", err)
	}
	secretID := os.Getenv("TENCENT_COS_SECRET_ID")
	secretKey := os.Getenv("TENCENT_COS_SECRET_KEY")
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("环境变量 TENCENT_COS_SECRET_ID/KEY 未设置")
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
	return client, nil
}

func cosPutObject(client *cos.Client, path string, data []byte) error {
	_, err := client.Object.Put(context.Background(), path, bytes.NewReader(data), nil)
	return err
}

func cosGetObject(client *cos.Client, path string) ([]byte, error) {
	resp, err := client.Object.Get(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ============================================================
// 主函数
// ============================================================

func main() {
	// ---- 解析参数 ----
	configPath := os.Getenv("JAPANESE_REVIEW_CONFIG")
	today := time.Now()

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		default:
			// 尝试当日期解析
			if t, err := parseDate(args[i]); err == nil {
				today = t
			}
		}
	}

	// ---- 加载配置 ----
	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 配置加载失败: %v\n", err)
		os.Exit(1)
	}
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// ---- 从COS同步最新数据 ----
	if cfg.COS.Enabled {
		client, err := newCosClient(cfg.COS.BucketURL, cfg.COS.Region)
		if err == nil {
			remoteData, err := cosGetObject(client, cfg.COS.Progress)
			if err == nil {
				// 验证是合法JSON后再覆盖本地
				var testDB VocabDB
				if json.Unmarshal(remoteData, &testDB) == nil && len(testDB.Words) > 0 {
					if err := os.WriteFile(cfg.ProgressFile, remoteData, 0644); err == nil {
						fmt.Println("  ☁️  已从COS同步最新数据")
					}
				}
			} else {
				fmt.Fprintf(os.Stderr, "  ⚠️  COS同步失败(使用本地数据): %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "  ⚠️  COS客户端创建失败(使用本地数据): %v\n", err)
		}
	}

	// ---- 读取数据库 ----
	data, err := os.ReadFile(cfg.ProgressFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 读取 %s 失败: %v\n", cfg.ProgressFile, err)
		os.Exit(1)
	}
	var db VocabDB
	if err := json.Unmarshal(data, &db); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 解析数据库失败: %v\n", err)
		os.Exit(1)
	}

	// ---- 计算到期单词 ----
	var dueItems []DueItem
	var redCount, yellowCount, greenCount int

	for _, w := range db.Words {
		if !isDue(cfg, w, today) {
			continue
		}
		cat := computeCategory(w)
		dueItems = append(dueItems, DueItem{
			Chinese:  w.Chinese,
			Japanese: w.Japanese,
			Category: cat,
		})
		switch cat {
		case "red":
			redCount++
		case "yellow":
			yellowCount++
		case "green":
			greenCount++
		}
	}

	// ---- 绿色抽样 ----
	var greens, others []DueItem
	for _, it := range dueItems {
		if it.Category == "green" {
			greens = append(greens, it)
		} else {
			others = append(others, it)
		}
	}
	if len(greens) > cfg.Green.MaxBeforeSample {
		// 按间隔排序（连续的长的优先抽查）
		// 这里简单排序，取间距最小的
		sort.Slice(greens, func(i, j int) bool {
			return greens[i].Seq < greens[j].Seq
		})
		sample := len(greens) / cfg.Green.SampleFraction
		if sample < cfg.Green.MinSample {
			sample = cfg.Green.MinSample
		}
		if sample > len(greens) {
			sample = len(greens)
		}
		greens = greens[:sample]
	}
	greenCount = len(greens)
	dueItems = append(others, greens...)

	// 重新编号
	for i := range dueItems {
		dueItems[i].Seq = i + 1
	}

	// ---- 造句随机抽取 ----
	var sentenceItems []DueItem
	pool := make([]Sentence, len(db.Sentences))
	copy(pool, db.Sentences)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	n := cfg.Sentences.DailyCount
	if n > len(pool) {
		n = len(pool)
	}
	for i, s := range pool[:n] {
		sentenceItems = append(sentenceItems, DueItem{
			Seq:      i + 1,
			Chinese:  s.Chinese,
			Japanese: s.Japanese,
			Category: "sentence",
		})
	}

	// ---- 生成 Excel ----
	dateStr := today.Format("20060102")
	outputFile := filepath.Join(cfg.OutputDir, fmt.Sprintf("日语复习_%s.xlsx", dateStr))
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建输出目录失败: %v\n", err)
		os.Exit(1)
	}
	if err := generateReviewExcel(cfg, dueItems, sentenceItems, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 生成Excel失败: %v\n", err)
		os.Exit(1)
	}

	// ---- 上传 COS ----
	if cfg.COS.Enabled {
		tryUploadToCOS(cfg, outputFile, fmt.Sprintf(cfg.COS.ReviewFmt, dateStr))
		tryUploadToCOS(cfg, cfg.ProgressFile, cfg.COS.Progress)
	}

	// ---- 输出 ----
	totalWords := redCount + yellowCount + greenCount
	fmt.Printf("✅ 日语复习列表已生成\n")
	fmt.Printf("   日期: %s\n", today.Format("2006-01-02"))
	fmt.Printf("   文件: %s\n", outputFile)
	fmt.Printf("   📊 今日到期: 🔴%d  🟡%d  🟢%d  📝%d  共%d项\n",
		redCount, yellowCount, greenCount, len(sentenceItems),
		totalWords+len(sentenceItems))
}
