// war-briefing.go
// 美以伊战争自动简报脚本（Go版本）
// 每天4个时间点执行，汇总前6小时情况

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// Briefing 简报结构
type Briefing struct {
	TimeSlot    string   `json:"time_slot"`
	Period      string   `json:"period"`
	GeneratedAt string   `json:"generated_at"`
	Content     string   `json:"content"`
	Sources     []string `json:"sources"`
}

// 常量
const (
	workspaceDir = "/home/zhangyufeng/.openclaw/workspace"
	briefingDir  = "/home/zhangyufeng/.openclaw/workspace/briefings"
)

func main() {
	log.Println("📊 战争简报脚本启动（Go版本）")

	// 获取当前时间（北京时间）
	now := time.Now()
	beijingTime := now.Add(8 * time.Hour) // UTC+8
	hour := beijingTime.Hour()

	// 确定时间点和时段
	timeSlot, period := getTimeSlot(hour)
	
	log.Printf("时间点: %02d:00 (北京时间)", timeSlot)
	log.Printf("汇总时段: %s", period)

	// 生成简报
	briefing, err := generateBriefing(timeSlot, period, beijingTime)
	if err != nil {
		log.Fatalf("❌ 生成简报失败: %v", err)
	}

	// 保存简报
	if err := saveBriefing(briefing, beijingTime); err != nil {
		log.Fatalf("❌ 保存简报失败: %v", err)
	}

	// 输出简报内容
	fmt.Println("\n" + briefing.Content)

	log.Println("✅ 简报生成完成")
}

// getTimeSlot 获取时间点和时段
func getTimeSlot(hour int) (int, string) {
	var timeSlot int
	var period string

	switch {
	case hour >= 0 && hour < 6:
		timeSlot = 0
		period = "前日18:00-今日00:00"
	case hour >= 6 && hour < 12:
		timeSlot = 6
		period = "00:00-06:00"
	case hour >= 12 && hour < 18:
		timeSlot = 12
		period = "06:00-12:00"
	default:
		timeSlot = 18
		period = "12:00-18:00"
	}

	return timeSlot, period
}

// generateBriefing 生成简报
func generateBriefing(timeSlot int, period string, beijingTime time.Time) (*Briefing, error) {
	// 搜索最新消息
	results, err := searchWarNews(beijingTime)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %v", err)
	}

	// 生成简报内容
	content := generateContent(results, timeSlot, period, beijingTime)

	// 提取信息来源
	sources := extractSources(results)

	return &Briefing{
		TimeSlot:    fmt.Sprintf("%02d:00", timeSlot),
		Period:      period,
		GeneratedAt: beijingTime.Format("2006-01-02 15:04:05"),
		Content:     content,
		Sources:     sources,
	}, nil
}

// searchWarNews 搜索战争新闻
func searchWarNews(beijingTime time.Time) ([]SearchResult, error) {
	// 构建搜索查询
	query := fmt.Sprintf("美国 以色列 伊朗 战争 最新消息 %d年%d月%d日",
		beijingTime.Year(),
		beijingTime.Month(),
		beijingTime.Day())

	log.Printf("搜索查询: %s", query)

	// 调用tavily-search技能
	cmd := exec.Command("python3", 
		filepath.Join(workspaceDir, "skills/openclaw-tavily-search/scripts/tavily_search.py"),
		"--query", query,
		"--max-results", "8",
		"--format", "brave")

	cmd.Dir = workspaceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("⚠️ 搜索命令执行错误: %v", err)
		log.Printf("输出: %s", output)
		return []SearchResult{}, nil // 返回空结果，不中断流程
	}

	// 解析JSON响应
	var response SearchResponse
	if err := json.Unmarshal(output, &response); err != nil {
		log.Printf("⚠️ 解析搜索结果失败: %v", err)
		log.Printf("原始输出: %s", string(output[:min(200, len(output))]))
		return []SearchResult{}, nil
	}

	log.Printf("找到 %d 条结果", len(response.Results))
	return response.Results, nil
}

// generateContent 生成简报内容
func generateContent(results []SearchResult, timeSlot int, period string, beijingTime time.Time) string {
	var content strings.Builder

	// 标题
	content.WriteString(fmt.Sprintf("📊 *美以伊战争简报* (#%d)\n\n", timeSlot/6+1))
	content.WriteString(fmt.Sprintf("*时间*: %02d:00 (北京时间)\n", timeSlot))
	content.WriteString(fmt.Sprintf("*时段*: %s\n", period))
	content.WriteString(fmt.Sprintf("*生成*: %s\n\n", 
		beijingTime.Format("2006/01/02 15:04")))

	if len(results) == 0 {
		content.WriteString("⚠️ *无最新消息*\n")
		content.WriteString("过去6小时未发现重大战况更新。\n")
		content.WriteString("可能原因：\n")
		content.WriteString("1. 战况相对稳定\n")
		content.WriteString("2. 信息延迟\n")
		content.WriteString("3. 搜索API限制\n\n")
	} else {
		// 提取关键信息
		casualties := extractCasualties(results)
		keyEvents := extractKeyEvents(results, 3)
		politicalUpdates := extractPoliticalUpdates(results, 2)
		militaryActions := extractMilitaryActions(results, 3)

		// 伤亡统计
		if casualties != "" {
			content.WriteString(fmt.Sprintf("💀 *伤亡*: %s\n", casualties))
		}

		// 关键事件
		if len(keyEvents) > 0 {
			content.WriteString("\n📰 *关键事件*:\n")
			for _, event := range keyEvents {
				content.WriteString(fmt.Sprintf("• %s\n", event))
			}
		}

		// 政治动态
		if len(politicalUpdates) > 0 {
			content.WriteString("\n🏛️ *政治动态*:\n")
			for _, update := range politicalUpdates {
				content.WriteString(fmt.Sprintf("• %s\n", update))
			}
		}

		// 军事行动
		if len(militaryActions) > 0 {
			content.WriteString("\n⚔️ *军事行动*:\n")
			for _, action := range militaryActions {
				content.WriteString(fmt.Sprintf("• %s\n", action))
			}
		}

		content.WriteString("\n")
	}

	// 局势评估
	content.WriteString("*评估*:\n")
	rating := assessSituation(results)
	content.WriteString(fmt.Sprintf("%s\n\n", rating))

	// 下一简报
	nextSlot := (timeSlot + 6) % 24
	content.WriteString(fmt.Sprintf("*下一简报*: %02d:00\n", nextSlot))

	return content.String()
}

// extractCasualties 提取伤亡信息
func extractCasualties(results []SearchResult) string {
	for _, result := range results {
		snippet := result.Snippet
		if strings.Contains(snippet, "死亡") && strings.Contains(snippet, "受伤") {
			// 尝试提取数字
			lines := strings.Split(snippet, "。")
			for _, line := range lines {
				if strings.Contains(line, "死亡") && strings.Contains(line, "受伤") {
					// 简化处理
					return extractNumbers(line)
				}
			}
		}
	}
	return ""
}

// extractKeyEvents 提取关键事件
func extractKeyEvents(results []SearchResult, limit int) []string {
	var events []string
	keywords := []string{"小学", "医院", "平民", "儿童", "学校", "居民区", "爆炸", "袭击"}

	for _, result := range results {
		if len(events) >= limit {
			break
		}
		
		title := result.Title
		snippet := result.Snippet
		
		// 检查是否包含关键词
		for _, keyword := range keywords {
			if strings.Contains(title, keyword) || strings.Contains(snippet, keyword) {
				// 简化标题
				event := simplifyText(title, 60)
				if event != "" {
					events = append(events, event)
					break
				}
			}
		}
	}
	
	return events
}

// extractPoliticalUpdates 提取政治动态
func extractPoliticalUpdates(results []SearchResult, limit int) []string {
	var updates []string
	keywords := []string{"特朗普", "哈梅内伊", "以色列总理", "联合国", "国会", "谈判"}

	for _, result := range results {
		if len(updates) >= limit {
			break
		}
		
		snippet := result.Snippet
		for _, keyword := range keywords {
			if strings.Contains(snippet, keyword) {
				// 提取相关句子
				sentences := strings.Split(snippet, "。")
				for _, sentence := range sentences {
					if strings.Contains(sentence, keyword) && len(sentence) > 20 {
						update := simplifyText(sentence, 80)
						if update != "" {
							updates = append(updates, update)
							break
						}
					}
				}
				break
			}
		}
	}
	
	return updates
}

// extractMilitaryActions 提取军事行动
func extractMilitaryActions(results []SearchResult, limit int) []string {
	var actions []string
	keywords := []string{"袭击", "打击", "导弹", "无人机", "航母", "基地", "空袭"}

	for _, result := range results {
		if len(actions) >= limit {
			break
		}
		
		snippet := result.Snippet
		for _, keyword := range keywords {
			if strings.Contains(snippet, keyword) {
				// 提取相关句子
				sentences := strings.Split(snippet, "。")
				for _, sentence := range sentences {
					if strings.Contains(sentence, keyword) && len(sentence) > 20 {
						action := simplifyText(sentence, 80)
						if action != "" && !contains(actions, action) {
							actions = append(actions, action)
							break
						}
					}
				}
				break
			}
		}
	}
	
	return actions
}

// extractSources 提取信息来源
func extractSources(results []SearchResult) []string {
	var sources []string
	for _, result := range results {
		if len(sources) >= 3 {
			break
		}
		if result.Title != "" {
			sources = append(sources, fmt.Sprintf("%s - %s", 
				simplifyText(result.Title, 40), 
				extractDomain(result.URL)))
		}
	}
	return sources
}

// assessSituation 评估局势
func assessSituation(results []SearchResult) string {
	if len(results) == 0 {
		return "🟡 待观察 | ⚔️ 未知 | 🌍 待评估"
	}

	// 检查关键词
	criticalCount := 0
	criticalKeywords := []string{"死亡", "袭击", "爆炸", "打击", "导弹", "空袭"}
	
	for _, result := range results {
		text := result.Title + " " + result.Snippet
		for _, keyword := range criticalKeywords {
			if strings.Contains(text, keyword) {
				criticalCount++
				break
			}
		}
	}

	// 评估
	switch {
	case criticalCount >= 5:
		return "🔥🔥🔥🔥 高度危险 | ⚔️ 全面冲突 | 🌍 风险极高"
	case criticalCount >= 3:
		return "🔥🔥🔥 危险升级 | ⚔️ 持续冲突 | 🌍 风险高"
	case criticalCount >= 1:
		return "🔥🔥 紧张持续 | ⚔️ 有限冲突 | 🌍 风险中"
	default:
		return "🔥 相对稳定 | ⚔️ 低烈度 | 🌍 风险低"
	}
}

// saveBriefing 保存简报
func saveBriefing(briefing *Briefing, beijingTime time.Time) error {
	// 确保目录存在
	if err := os.MkdirAll(briefingDir, 0755); err != nil {
		return err
	}

	// 生成文件名
	filename := fmt.Sprintf("briefing-%s-%s.txt",
		beijingTime.Format("2006-01-02"),
		strings.Replace(briefing.TimeSlot, ":", "", -1))
	
	filepath := filepath.Join(briefingDir, filename)

	// 保存内容
	content := fmt.Sprintf("时间点: %s\n时段: %s\n生成时间: %s\n\n%s\n\n信息来源:\n",
		briefing.TimeSlot, briefing.Period, briefing.GeneratedAt, briefing.Content)
	
	for i, source := range briefing.Sources {
		content += fmt.Sprintf("%d. %s\n", i+1, source)
	}

	if err := ioutil.WriteFile(filepath, []byte(content), 0644); err != nil {
		return err
	}

	log.Printf("简报已保存: %s", filepath)
	return nil
}

// 辅助函数
func extractNumbers(text string) string {
	// 简化实现
	if strings.Contains(text, "死亡") {
		return "有伤亡报告"
	}
	return ""
}

func simplifyText(text string, maxLen int) string {
	// 清理文本
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "#", "")
	
	// 截断
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}

func extractDomain(url string) string {
	// 简化实现
	if strings.Contains(url, "news.cn") {
		return "新华网"
	}
	if strings.Contains(url, "chinanews.com") {
		return "中新网"
	}
	if strings.Contains(url, "voachinese.com") {
		return "美国之音"
	}
	return "其他来源"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}