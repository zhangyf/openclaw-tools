// Daily Fortune — 每日运势生成 + 三択占い + COS归档 + 预签名URL
// 用法：
//   daily-fortune                        → 今天的运势（默认生日1990-06-15 JST）
//   daily-fortune 1984 10 17             → 指定生日的今日运势
// 输出：
//   1. 详细运势 + 三択占い 合并文本 → 上传COS
//   2. 打印预签名URL（1小时有效）
//
// 依赖：DEEPSEEK_API_KEY + TENCENT_COS_SECRET_{ID,KEY}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// ============================================================
// 配置
// ============================================================

const (
	apiEndpoint = "https://api.deepseek.com/chat/completions"
	bucketURL   = "https://openclaw-backup-tx-1251036673.cos.ap-beijing.myqcloud.com"
)

var targetDate time.Time

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

// getDeepSeekKey 从环境变量或 OpenClaw 配置获取 DeepSeek API Key
func getDeepSeekKey() string {
	if k := os.Getenv("DEEPSEEK_API_KEY"); k != "" {
		return k
	}
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
	for _, name := range []string{"deepseek", "deepseek-v4"} {
		if p, ok := cfg.Models.Providers[name]; ok && p.ApiKey != "" {
			return p.ApiKey
		}
	}
	return ""
}

// askDeepSeek 调用 DeepSeek API 生成文本
func askDeepSeek(system, user string) (string, error) {
	key := getDeepSeekKey()
	if key == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY 未设置")
	}
	body, _ := json.Marshal(chatReq{
		Model:       "deepseek-chat",
		Messages:    []chatMsg{{Role: "system", Content: system}, {Role: "user", Content: user}},
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
// COS操作（Go SDK）
// ============================================================

// getCOSClient 创建腾讯云COS客户端
func getCOSClient() *cos.Client {
	secretID := os.Getenv("TENCENT_COS_SECRET_ID")
	secretKey := os.Getenv("TENCENT_COS_SECRET_KEY")
	if secretID == "" || secretKey == "" {
		fmt.Fprintln(os.Stderr, "错误: TENCENT_COS_SECRET_ID/KEY 未设置")
		os.Exit(1)
	}
	u, _ := url.Parse(bucketURL)
	b := &cos.BaseURL{BucketURL: u}
	return cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
}

// cosUpload 上传内容到COS
func cosUpload(client *cos.Client, content []byte, cosPath string) error {
	_, err := client.Object.Put(context.Background(), cosPath, bytes.NewReader(content), nil)
	if err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	fmt.Printf("  ☁️  已上传COS: openclaw-backup-tx-1251036673/%s\n", cosPath)
	return nil
}

// generatePresignedURL 生成COS预签名下载URL（指定过期秒数）
func generatePresignedURL(client *cos.Client, cosPath string, expireSeconds int) (string, error) {
	secretID := os.Getenv("TENCENT_COS_SECRET_ID")
	secretKey := os.Getenv("TENCENT_COS_SECRET_KEY")
	presignedURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodGet,
		cosPath,
		secretID,
		secretKey,
		time.Duration(expireSeconds)*time.Second,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %w", err)
	}
	return presignedURL.String(), nil
}

// ============================================================
// 生成图提示词
// ============================================================

func generateImagePrompt(birthYear, birthMonth, birthDay int) (string, error) {
	dateStrJP := fmt.Sprintf("%d/%d(%s)", targetDate.Month(), targetDate.Day(), weekDayJP(targetDate))
	prompt := fmt.Sprintf(`あなたは日本の占い画像生成のプロンプトを作るアシスタントです。
以下の情報をもとに、AI画像生成用の英語プロンプトを作成してください。

【ユーザー生年月日】%d年%d月%d日
【対象日】%s

プロンプト要件：
- 日本語の四柱推命（フォーチュンテリング）をテーマにした画像
- 和風テイスト、かわいくて親しみやすい雰囲気
- 星座・星・花・和柄などの装飾要素を含む
- ラッキーカラーをアクセントカラーとして取り入れる
- ソーシャルメディア（X/Twitter）投稿向け、900x900pxの正方形
- テキストなし、グラフィックのみ
- 出力は英語のプロンプト文のみ（100語以内）`, birthYear, birthMonth, birthDay, dateStrJP)

	system := `あなたはプロの画像生成プロンプトライター。
要件：
1. 英語で100語以内
2. 日本語占いテーマ、和風かわいい系
3. 絵文字や特殊文字は使わない
4. プロンプトのみを出力、説明不要
5. Midjourney / DALL-E / Stable Diffusion いずれでも使える汎用的な形式`
	return askDeepSeek(system, prompt)
}

// ============================================================
// 合并内容
// ============================================================

func buildCombinedContent(detailed, tweet3, imgPrompt string) string {
	dateLabel := fmt.Sprintf("%s（%s）", targetDate.Format("2006年1月2日"), weekDayJP(targetDate))
	var b strings.Builder
	b.WriteString("========================================\n")
	b.WriteString(fmt.Sprintf("  📅 %s\n", dateLabel))
	b.WriteString(fmt.Sprintf("  🏷️  fortune/%s/%s.txt\n", targetDate.Format("2006/01"), targetDate.Format("2006-01-02")))
	b.WriteString("========================================\n\n")
	b.WriteString("【Part 1】🌸 今日の詳細運勢\n")
	b.WriteString("──────────────────────────\n")
	b.WriteString(detailed)
	b.WriteString("\n\n")
	b.WriteString("【Part 2】🎯 三択占い\n")
	b.WriteString("──────────────────────────\n")
	b.WriteString(tweet3)
	b.WriteString("\n\n")
	b.WriteString("【Part 3】🎨 生図プロンプト（画像生成用）\n")
	b.WriteString("──────────────────────────\n")
	b.WriteString(imgPrompt)
	b.WriteString("\n\n")
	b.WriteString("========================================\n")
	b.WriteString(fmt.Sprintf("  生成時刻: %s (CST)\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("========================================\n")
	return b.String()
}

// ============================================================
// 入口
// ============================================================

func main() {
	fmt.Println("🔮 每日运势生成 + 三択占い + COS归档")
	fmt.Println(strings.Repeat("━", 40))

	// 参数解析：可指定生日（默认1990-06-15 JST）
	birthYear, birthMonth, birthDay := 1990, 6, 15
	if len(os.Args) >= 4 {
		birthYear, _ = strconv.Atoi(os.Args[1])
		birthMonth, _ = strconv.Atoi(os.Args[2])
		birthDay, _ = strconv.Atoi(os.Args[3])
	}

	targetDate = time.Now()
	dateStr := targetDate.Format("2006-01-02")
	cosPath := fmt.Sprintf("fortune/%s/%s.txt", targetDate.Format("2006/01"), dateStr)
	dateJP := fmt.Sprintf("%d/%d(%s)", targetDate.Month(), targetDate.Day(), weekDayJP(targetDate))

	fmt.Printf("   生日: %d/%d/%d\n", birthYear, birthMonth, birthDay)
	fmt.Printf("   対象: %s\n", dateJP)
	fmt.Printf("   COS: openclaw-backup-tx-1251036673/%s\n", cosPath)
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
	fmt.Println("✅")

	// Step 2.5: 生成图提示词
	fmt.Print("  🎨 生成图提示词... ")
	imgPrompt, err := generateImagePrompt(birthYear, birthMonth, birthDay)
	if err != nil {
		fmt.Printf("⚠️  %v\n", err)
		imgPrompt = "[生成失败]"
	} else {
		fmt.Println("✅")
	}

	// Step 3: 合并内容
	fmt.Print("  📦 合并内容... ")
	combined := buildCombinedContent(detailed, tweet3, imgPrompt)
	fmt.Printf("✅ (%d文字)\n", len([]rune(combined)))

	// Step 4: 初始化COS客户端
	fmt.Print("  🔑 初始化COS... ")
	client := getCOSClient()
	fmt.Println("✅")

	// Step 5: 上传COS
	fmt.Print("  ☁️  上传COS... ")
	if err := cosUpload(client, []byte(combined), cosPath); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Print("  ✅\n")

	// Step 6: 生成预签名URL
	fmt.Print("  🔗 生成预签名URL... ")
	presignedURL, err := generatePresignedURL(client, cosPath, 3600) // 1小时有效
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅")

	fmt.Println(strings.Repeat("━", 40))
	fmt.Println("\n  📎 预签名URL（1小时有效）:\n")
	fmt.Printf("  %s\n\n", presignedURL)
	fmt.Println(strings.Repeat("━", 40))
	fmt.Println("\n  🎨 画像生成プロンプト（用于其他工具）:\n")
	fmt.Printf("  %s\n\n", imgPrompt)
	fmt.Println(strings.Repeat("━", 40))
}
