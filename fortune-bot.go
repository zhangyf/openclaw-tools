// 占い自動投稿ジェネレーター
// Fortune Bot — 每日运势生成 + COS归档（通过Python SDK上传）
// 用法：
//   fortune-bot                        → 今天的运势（默认生日1990-06-15）
//   fortune-bot 1984 10 17             → 指定生日的今日运势
//   fortune-bot --upload 1984 10 17    → 生成并上传COS
//   fortune-bot --batch 7 1984 10 17   → 批量生成7天
// 依赖：DEEPSEEK_API_KEY + TENCENT_COS_SECRET_{ID,KEY}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const apiEndpoint = "https://api.deepseek.com/chat/completions"

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

func askDeepSeek(system, user string) (string, error) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}
	body, _ := json.Marshal(chatReq{
		Model: "deepseek-chat",
		Messages: []chatMsg{
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
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var cr chatResp
	json.Unmarshal(raw, &cr)
	if cr.Error != nil {
		return "", fmt.Errorf(cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("空响应")
	}
	return cr.Choices[0].Message.Content, nil
}

// ============================================================
// COS上传（通过Python SDK）
// ============================================================

func cosUpload(content []byte, cosPath string) error {
	// 调用cos-upload.py脚本上传
	cmd := exec.Command("python3", "cos-upload.py", cosPath)
	cmd.Stdin = bytes.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func uploadFortune(content string, t time.Time) error {
	dir := t.Format("fortune/2006/01")
	cosPath := dir + "/" + t.Format("2006-01-02") + ".txt"
	return cosUpload([]byte(content+"\n"), cosPath)
}

// ============================================================
// 运势生成
// ============================================================

func dateJP(t time.Time) string {
	weekdays := []string{"日", "月", "火", "水", "木", "金", "土"}
	return fmt.Sprintf("%d/%d(%s)", t.Month(), t.Day(), weekdays[t.Weekday()])
}

func generateFortune(birthYear, birthMonth, birthDay int, targetDate time.Time) (string, error) {
	dateStr := dateJP(targetDate)
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
		birthYear, birthMonth, birthDay, dateStr, dateStr)

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

func generateBatch(birthYear, birthMonth, birthDay, days int) ([]string, error) {
	var results []string
	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, i)
		content, err := generateFortune(birthYear, birthMonth, birthDay, date)
		if err != nil {
			return results, fmt.Errorf("%s: %w", dateJP(date), err)
		}
		results = append(results, content)
		if i < days-1 {
			time.Sleep(2 * time.Second)
		}
	}
	return results, nil
}

// ============================================================
// 入口
// ============================================================

func main() {
	fmt.Println("🔮 占い自動投稿ジェネレーター")
	fmt.Println(strings.Repeat("━", 40))

	birthYear, birthMonth, birthDay := 1990, 6, 15
	mode := "single"
	batchDays := 7
	doUpload := false

	args := os.Args[1:]
	var dateArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--batch", "-b":
			mode = "batch"
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					batchDays = n
					i++
				}
			}
		case "--upload", "-u":
			doUpload = true
		default:
			dateArgs = append(dateArgs, args[i])
		}
	}
	if len(dateArgs) >= 3 {
		birthYear, _ = strconv.Atoi(dateArgs[0])
		birthMonth, _ = strconv.Atoi(dateArgs[1])
		birthDay, _ = strconv.Atoi(dateArgs[2])
	}

	fmt.Printf("  誕生日: %d年%d月%d日\n", birthYear, birthMonth, birthDay)
	if mode == "batch" {
		fmt.Printf("  批量生成: %d日分\n", batchDays)
	}
	if doUpload {
		fmt.Println("  上传COS: ✓")
	}
	fmt.Println(strings.Repeat("━", 40))

	now := time.Now()

	if mode == "batch" {
		fmt.Println("\n  生成中...\n")
		contents, err := generateBatch(birthYear, birthMonth, birthDay, batchDays)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}
		for i, c := range contents {
			date := now.AddDate(0, 0, i)
			filename := fmt.Sprintf("fortune_%s.txt", date.Format("2006-01-02"))
			os.WriteFile(filename, []byte(c+"\n"), 0644)
			fmt.Printf("  ✅ %s → %s\n", dateJP(date), filename)
			if doUpload {
				if err := uploadFortune(c, date); err != nil {
					fmt.Fprintf(os.Stderr, "  ⚠️  上传失败: %v\n", err)
				}
			}
		}
		fmt.Printf("\n  ✅ %d日分完成\n", len(contents))
	} else {
		fmt.Println("\n  生成中...\n")
		content, err := generateFortune(birthYear, birthMonth, birthDay, now)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(content)
		fmt.Println(strings.Repeat("━", 40))
		fmt.Printf("  文字数: %d\n", len([]rune(content)))

		filename := fmt.Sprintf("fortune_%s.txt", now.Format("2006-01-02"))
		os.WriteFile(filename, []byte(content+"\n"), 0644)
		fmt.Printf("  💾 已保存到: %s\n", filename)

		if doUpload {
			if err := uploadFortune(content, now); err != nil {
				fmt.Fprintf(os.Stderr, "  ⚠️  COS上传失败: %v\n", err)
			}
		}
	}
}
