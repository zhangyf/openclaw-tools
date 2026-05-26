// Daily Fortune — 每日运势生成 + 三択占い + COS归档 + 预签名URL
// 用法：
//   go run daily-fortune.go                       → 今天的运势（默认生日1990-06-15 JST）
//   go run daily-fortune.go 1984 10 17             → 指定生日的今日运势
// 输出：
//   1. 详细运势 + 三択占い 合并文本 → 上传COS
//   2. 打印预签名URL（1小时有效）

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 配置
// ============================================================

const apiEndpoint = "https://api.deepseek.com/chat/completions"

var (
	bucket   = "openclaw-backup-tx-1251036673"
	region   = "ap-beijing"
	cosDir   string // fortune/YYYY/MM
	cosPath  string // fortune/YYYY/MM/YYYY-MM-DD.txt
	dateStr  string // YYYY-MM-DD
	dateJP   string // M/D(火)形式
	targetDate time.Time
)

// ============================================================
// DeepSeek API
// ============================================================

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type chatReq struct {
	Model       string    `json:"model"`
	Messages    []chatMsg `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}
type chatResp struct {
	Choices []struct {
		Message chatMsg `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func getDeepSeekKey() string {
	// 优先环境变量，其次从 OpenClaw 配置文件读取
	if k := os.Getenv("DEEPSEEK_API_KEY"); k != "" {
		return k
	}
	// 从 openclaw.json 读取
	cfgPath := filepath.Join(os.Getenv("HOME"), ".openclaw", "openclaw.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return ""
	}
	var cfg struct {
		Models struct {
			Providers map[string]struct {
				ApiKey string `json:"apiKey"`
			} `json:"providers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	if p, ok := cfg.Models.Providers["deepseek"]; ok && p.ApiKey != "" {
		return p.ApiKey
	}
	if p, ok := cfg.Models.Providers["deepseek-v4"]; ok && p.ApiKey != "" {
		return p.ApiKey
	}
	return ""
}

func askDeepSeek(system, user string) (string, error) {
	key := getDeepSeekKey()
	if key == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}
	body, _ := json.Marshal(chatReq{
		Model:       "deepseek-chat",
		Messages:    []chatMsg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.7,
		MaxTokens:   800,
	})
	req, _ := http.NewRequest("POST", apiEndpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var cr chatResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("API错误: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("空响应")
	}
	return cr.Choices[0].Message.Content, nil
}

// ============================================================
// 运势生成（详细版）
// ============================================================

func weekDayJP(t time.Time) string {
	days := []string{"日", "月", "火", "水", "木", "金", "土"}
	return days[t.Weekday()]
}

func generateDetailedFortune(birthYear, birthMonth, birthDay int) (string, error) {
	dateStrJP := fmt.Sprintf("%d/%d(%s)", targetDate.Month(), targetDate.Day(), weekDayJP(targetDate))

	prompt := fmt.Sprintf(`以下の情報で今日の四柱推命占いを作成してください。

【ユーザー生年月日】%d年%d月%d日
【対象日】%s

出力は以下のフォーマットで、100〜130文字以内でお願いします：

%s🌸今日の運勢
⭐️★★★☆☆ （一言で）

💼仕事→（一言）
💕恋愛→（一言）
🌈LC：（色）
🔢LN：（数字）
🧭方角：（方位）

「一言メッセージ」

#今日の運勢`,
		birthYear, birthMonth, birthDay, dateStrJP, dateStrJP)

	system := `あなたは人気占い系Xアカウント。毎日かわいくて親しみやすい運勢を発信。

ルール：
1. 100〜130文字に収める（厳守）
2. 各項目はemojiで飾る（🌸⭐️💼💕🌈🔢🧭💬）
3. 語尾は「だよ」「〜ね」「！」など軽め・かわいめに
4. ポジティブ中心。悪いことも柔らかく包む
5. ラッキーカラー・ナンバー・方位は必ず入れる
6. 最後に #今日の運勢 タグ`

	return askDeepSeek(system, prompt)
}

// ============================================================
// 三択占い生成（sizhu --tweet）
// ============================================================

func generate3ChoiceTweet() string {
	sizhuPath := filepath.Join(os.Getenv("HOME"), ".openclaw", "workspace", "skills", "sizhu", "sizhu")
	cmd := exec.Command(sizhuPath, "--tweet", "9") // 9 = JST
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("[三択生成失敗: %v]", err)
	}
	return strings.TrimSpace(string(output))
}

// ============================================================
// COS上传（通过Python SDK）
// ============================================================

func cosUpload(content string) error {
	scriptPath := filepath.Join(os.Getenv("HOME"), ".openclaw", "workspace", "cos-upload.py")
	cmd := exec.Command("python3", scriptPath, cosPath)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	fmt.Print(string(out))
	return nil
}

// ============================================================
// 预签名URL生成（通过Python SDK）
// ============================================================

func generatePresignedURL(expireSeconds int) (string, error) {
	scriptPath := filepath.Join(os.Getenv("HOME"), ".openclaw", "workspace", "cos-presign.py")
	cmd := exec.Command("python3", scriptPath, cosPath, fmt.Sprintf("%d", expireSeconds))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("预签名URL生成失败: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ============================================================
// 合并内容
// ============================================================

func buildCombinedContent(detailed, tweet3 string) string {
	dateLabel := fmt.Sprintf("%s（%s）", targetDate.Format("2006年1月2日"), weekDayJP(targetDate))
	var b strings.Builder
	b.WriteString("========================================\n")
	b.WriteString(fmt.Sprintf("  📅 %s\n", dateLabel))
	b.WriteString(fmt.Sprintf("  🏷️  %s\n", cosPath))
	b.WriteString("========================================\n\n")

	b.WriteString("【Part 1】🌸 今日の詳細運勢\n")
	b.WriteString("──────────────────────────\n")
	b.WriteString(detailed)
	b.WriteString("\n\n")

	b.WriteString("【Part 2】🎯 三択占い\n")
	b.WriteString("──────────────────────────\n")
	b.WriteString(tweet3)
	b.WriteString("\n\n")

	b.WriteString("========================================\n")
	b.WriteString(fmt.Sprintf("  生成時刻: %s (JST)\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("========================================\n")

	return b.String()
}

// ============================================================
// 入口
// ============================================================

func main() {
	fmt.Println("🔮 每日运势生成 + 三択占い + COS归档")
	fmt.Println(strings.Repeat("━", 40))

	// 参数解析
	birthYear, birthMonth, birthDay := 1990, 6, 15
	if len(os.Args) >= 4 {
		birthYear, _ = strconv.Atoi(os.Args[1])
		birthMonth, _ = strconv.Atoi(os.Args[2])
		birthDay, _ = strconv.Atoi(os.Args[3])
	}

	// 目标日期（今天）
	targetDate = time.Now()
	dateStr = targetDate.Format("2006-01-02")
	cosDir = targetDate.Format("fortune/2006/01")
	cosPath = cosDir + "/" + targetDate.Format("2006-01-02") + ".txt"
	dateJP = fmt.Sprintf("%d/%d(%s)", targetDate.Month(), targetDate.Day(), weekDayJP(targetDate))

	fmt.Printf("   生日: %d/%d/%d\n", birthYear, birthMonth, birthDay)
	fmt.Printf("   対象: %s\n", dateJP)
	fmt.Printf("   COS: %s/%s\n", bucket, cosPath)
	fmt.Println(strings.Repeat("━", 40))

	// Step 1: 生成详细运势
	fmt.Print("\n  📝 生成详细运势... ")
	detailed, err := generateDetailedFortune(birthYear, birthMonth, birthDay)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ (%d文字)\n", len([]rune(detailed)))

	// Step 2: 生成三択占い
	fmt.Print("  🎯 生成三択占い... ")
	tweet3 := generate3ChoiceTweet()
	fmt.Printf("✅\n")

	// Step 3: 合并内容
	fmt.Print("  📦 合并内容... ")
	combined := buildCombinedContent(detailed, tweet3)
	fmt.Printf("✅ (%d文字)\n", len([]rune(combined)))

	// Step 4: 上传COS
	fmt.Print("  ☁️  上传COS... ")
	if err := cosUpload(combined); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Print("  ✅ 上传完成\n")

	// Step 5: 生成预签名URL
	fmt.Print("  🔗 生成预签名URL... ")
	presignedURL, err := generatePresignedURL(3600) // 1小时有效
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅")

	fmt.Println(strings.Repeat("━", 40))
	fmt.Println("\n  📎 预签名URL（1小时有效）:\n")
	fmt.Printf("  %s\n\n", presignedURL)
	fmt.Println(strings.Repeat("━", 40))
}
