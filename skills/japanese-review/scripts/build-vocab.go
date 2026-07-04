// build-vocab.go — 从日语学习进度档案 Markdown 提取词汇数据，生成 progress.json
// 一次性工具，后续只需运行 daily-review.go

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Word 表示一个单词的学习数据
type Word struct {
	Japanese          string `json:"japanese"`
	Chinese           string `json:"chinese"`
	ReviewCount       int    `json:"review_count"`
	ErrorCount        int    `json:"error_count"`
	ConsecutiveCorrect int   `json:"consecutive_correct"`
	LastReview        string `json:"last_review"` // MM/DD 格式
	Status            string `json:"status"`      // green/yellow/red/pending
}

// Sentence 表示造句练习
type Sentence struct {
	ID                string `json:"id"`
	Chinese           string `json:"chinese"`
	Japanese          string `json:"japanese"`
	ReviewCount       int    `json:"review_count"`
	ErrorCount        int    `json:"error_count"`
	ConsecutiveCorrect int   `json:"consecutive_correct"`
	LastReview        string `json:"last_review"`
	Status            string `json:"status"`
}

// VocabDB 是整个词汇数据库
type VocabDB struct {
	Words     []Word     `json:"words"`
	Sentences []Sentence `json:"sentences"`
	Updated   string     `json:"updated"`
}

func main() {
	markdownPath := "/home/zhangyufeng/.openclaw/media/inbound/日语学习进度档案_0704_20260704_101748---b229df66-bfab-49c0-aa5a-89bc4bfec80d"
	outPath := "/home/zhangyufeng/.openclaw/workspace/progress.json"

	data, err := os.ReadFile(markdownPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取文件失败: %v\n", err)
		os.Exit(1)
	}

	content := string(data)
	db := extractVocab(content)
	db.Updated = time.Now().Format("2006-01-02")

	jsonData, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON序列化失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 提取完成: %d 个单词, %d 个造句\n", len(db.Words), len(db.Sentences))
	fmt.Printf("   已保存到: %s\n", outPath)
}

// statusNormalize 将中文状态转为英文标识
// 注意：必须优先检查🔄(待测试)，因为其描述可能同时包含其他关键词
func statusNormalize(s string) string {
	s = strings.TrimSpace(s)
	// 优先级1: 待测试（🔄）优先——未测过的不算已掌握
	if strings.Contains(s, "🔄") || strings.Contains(s, "待测试") {
		return "pending"
	}
	// 优先级2: 待巩固
	if strings.Contains(s, "🔴") || strings.Contains(s, "待巩固") {
		return "red"
	}
	// 优先级3: 基本掌握
	if strings.Contains(s, "🟡") || strings.Contains(s, "基本掌握") {
		return "yellow"
	}
	// 优先级4: 已掌握
	if strings.Contains(s, "🟢") || strings.Contains(s, "已掌握") {
		return "green"
	}
	return "pending"
}

func extractVocab(content string) VocabDB {
	var db VocabDB

	// 匹配所有表格行: |日语|中文|复习次数|答错次数|连续对|上次复习|状态|
	// 兼容有无空格、是否有对齐线
	re := regexp.MustCompile(`^\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|\s*([\d/]+|—|-)\s*\|\s*([^|]+?)\s*\|$`)

	// 匹配造句表格行: |S\d+|中文|日语|... (可能只有前4列)
	sentRe := regexp.MustCompile(`^\|(S\d+)\|([^|]+)\|([^|]*)\|`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNo := 0
	seen := make(map[string]bool) // 去重

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNo++

		// 跳过表头行和分隔行
		if strings.HasPrefix(line, "|---") || strings.Contains(line, "日语") && strings.Contains(line, "中文") && strings.Contains(line, "复习次数") {
			continue
		}

		// 匹配单词表行
		if m := re.FindStringSubmatch(line); m != nil {
			jp := strings.TrimSpace(m[1])
			cn := strings.TrimSpace(m[2])

			// 去重
			key := jp + "|" + cn
			if seen[key] {
				continue
			}
			seen[key] = true

			// 跳过明显不是单词的行（如复习记录描述）
			if strings.Contains(cn, "复习") || strings.Contains(cn, "记录") || len(cn) > 30 {
				continue
			}

			revCnt, _ := strconv.Atoi(strings.TrimSpace(m[3]))
			errCnt, _ := strconv.Atoi(strings.TrimSpace(m[4]))
			consec, _ := strconv.Atoi(strings.TrimSpace(m[5]))
			lastRev := strings.TrimSpace(m[6])
			if lastRev == "—" || lastRev == "-" {
				lastRev = ""
			}
			status := statusNormalize(m[7])

			// 修复一些特殊标记
			if status == "pending" && errCnt > 0 {
				status = "red"
			}
			if status == "pending" && revCnt >= 5 && errCnt == 0 {
				status = "green"
			}
			if status == "pending" && revCnt >= 1 && errCnt == 0 {
				status = "yellow"
			}

			db.Words = append(db.Words, Word{
				Japanese:          jp,
				Chinese:           cn,
				ReviewCount:       revCnt,
				ErrorCount:        errCnt,
				ConsecutiveCorrect: consec,
				LastReview:        lastRev,
				Status:            status,
			})
			continue
		}

		// 匹配造句行
		if m := sentRe.FindStringSubmatch(line); m != nil {
			id := strings.TrimSpace(m[1])
			cn := strings.TrimSpace(m[2])
			jp := strings.TrimSpace(m[3])

			if seen["S:"+id] {
				continue
			}
			seen["S:"+id] = true

			db.Sentences = append(db.Sentences, Sentence{
				ID:      id,
				Chinese: cn,
				Japanese: jp,
				Status:  "yellow",
			})
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "扫描错误: %v\n", err)
	}

	return db
}
