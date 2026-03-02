// war-briefing-go.go
// 战争财经简报脚本（Go版本）
// 激进模式：有消息就报，你去求证

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

// Briefing 简报结构
type Briefing struct {
	Title     string    `json:"title"`
	Time      string    `json:"time"`
	Period    string    `json:"period"`
	Generated string    `json:"generated"`
	War       WarInfo   `json:"war"`
	Finance   Finance   `json:"finance"`
	Strategy  []string  `json:"strategy"`
	Assessment Assessment `json:"assessment"`
	NextBrief string    `json:"next_brief"`
}

// WarInfo 战争信息
type WarInfo struct {
	HamaneyStatus string `json:"hamaney_status"`
	HamaneySource string `json:"hamaney_source,omitempty"`
	HormuzStatus  string `json:"hormuz_status"`
	HormuzSource  string `json:"hormuz_source,omitempty"`
	KeyEvent      string `json:"key_event,omitempty"`
}

// Finance 财经信息
type Finance struct {
	OilPrice    string `json:"oil_price"`
	OilSource   string `json:"oil_source,omitempty"`
	ChinaStock  string `json:"china_stock"`
	HKStock     string `json:"hk_stock"`
	SafeAssets  string `json:"safe_assets"`
	KeyFinance  string `json:"key_finance,omitempty"`
}

// Assessment 市场评估
type Assessment struct {
	Risk      string `json:"risk"`
	Stock     string `json:"stock"`
	Oil       string `json:"oil"`
	Advice    string `json:"advice"`
}

func main() {
	log.Println("📊 生成战争财经简报（Go版本）")
	
	// 确定时段
	now := time.Now()
	beijing := now.In(time.FixedZone("CST", 8*3600))
	
	var period string
	var nextBrief string
	
	switch beijing.Hour() {
	case 0, 1, 2, 3, 4, 5:
		period = "前日18:00-今日00:00"
		nextBrief = "6:00"
	case 6, 7, 8, 9, 10, 11:
		period = "00:00-06:00"
		nextBrief = "12:00"
	case 12, 13, 14, 15, 16, 17:
		period = "06:00-12:00"
		nextBrief = "18:00"
	default:
		period = "12:00-18:00"
		nextBrief = "24:00"
	}
	
	timeStr := fmt.Sprintf("%d:00", beijing.Hour())
	
	log.Printf("时间: %s (北京时间)", timeStr)
	log.Printf("时段: %s", period)
	
	// 搜索战争新闻
	warQuery := "伊朗官方确认哈梅内伊 死亡 最新消息 2026年3月2日 霍尔木兹海峡 封锁"
	warResults := searchTavily(warQuery, 3)
	
	// 搜索财经新闻
	financeQuery := "石油价格 暴涨 霍尔木兹海峡 封锁 股市 影响 2026年3月2日"
	financeResults := searchTavily(financeQuery, 3)
	
	// 生成简报
	briefing := generateBriefing(timeStr, period, nextBrief, warResults, financeResults)
	
	// 保存简报
	saveBriefing(briefing)
	
	// 输出简报
	printBriefing(briefing)
	
	log.Println("✅ 简报生成完成")
}

// searchTavily 搜索Tavily API
func searchTavily(query string, maxResults int) []SearchResult {
	log.Printf("搜索: %s", query)
	
	// 调用Python脚本（保持兼容）
	cmd := exec.Command("python3", 
		"/home/zhangyufeng/.openclaw/workspace/skills/openclaw-tavily-search/scripts/tavily_search.py",
		"--query", query,
		"--max-results", fmt.Sprintf("%d", maxResults),
		"--format", "brave")
	
	output, err := cmd.Output()
	if err != nil {
		log.Printf("⚠️ 搜索失败: %v", err)
		return []SearchResult{}
	}
	
	var data struct {
		Results []SearchResult `json:"results"`
	}
	
	if err := json.Unmarshal(output, &data); err != nil {
		log.Printf("⚠️ 解析搜索结果失败: %v", err)
		return []SearchResult{}
	}
	
	log.Printf("找到 %d 个结果", len(data.Results))
	return data.Results
}

// generateBriefing 生成简报
func generateBriefing(timeStr, period, nextBrief string, warResults, financeResults []SearchResult) *Briefing {
	briefing := &Briefing{
		Title:     fmt.Sprintf("美以伊战争财经简报 (#%d)", getBriefingNumber()),
		Time:      timeStr,
		Period:    period,
		Generated: time.Now().Format("2006/01/02 15:04"),
		NextBrief: nextBrief,
	}
	
	// 战争动态
	briefing.War = analyzeWar(warResults)
	
	// 财经影响
	briefing.Finance = analyzeFinance(financeResults)
	
	// 投资策略
	briefing.Strategy = []string{
		"减仓航空、航运股",
		"关注石油、黄金板块",
		"军工股短期机会",
		"新能源替代逻辑",
		"控制仓位，谨慎抄底",
	}
	
	// 市场评估
	briefing.Assessment = Assessment{
		Risk:   "高度危险",
		Stock:  "股市看跌",
		Oil:    "石油看涨",
		Advice: "大幅减仓，等待局势明朗",
	}
	
	return briefing
}

// analyzeWar 分析战争信息
func analyzeWar(results []SearchResult) WarInfo {
	war := WarInfo{
		HamaneyStatus: "暂无最新消息",
		HormuzStatus:  "状态正常",
	}
	
	// 哈梅内伊状态
	for _, result := range results {
		fullText := result.Title + " " + result.Snippet
		
		if strings.Contains(fullText, "哈梅内伊") {
			// 激进模式：有死亡相关词就报
			deathKeywords := []string{"身亡", "死亡", "遇难", "逝世", "去世", "被杀"}
			confirmKeywords := []string{"确认", "证实", "宣布", "承认"}
			denyKeywords := []string{"否认", "安全", "活着", "假消息"}
			
			hasDeath := false
			hasConfirm := false
			hasDeny := false
			
			for _, kw := range deathKeywords {
				if strings.Contains(fullText, kw) {
					hasDeath = true
					break
				}
			}
			
			for _, kw := range confirmKeywords {
				if strings.Contains(fullText, kw) {
					hasConfirm = true
					break
				}
			}
			
			for _, kw := range denyKeywords {
				if strings.Contains(fullText, kw) {
					hasDeny = true
					break
				}
			}
			
			if hasDeath {
				if hasConfirm {
					war.HamaneyStatus = "✅ 伊朗官方确认身亡"
				} else if hasDeny {
					war.HamaneyStatus = "❌ 伊朗否认身亡（据称安全）"
				} else {
					war.HamaneyStatus = "⚠️ 据报身亡（待你求证）"
				}
				war.HamaneySource = truncate(result.Title, 60)
				break
			}
		}
		
		// 霍尔木兹海峡
		if strings.Contains(fullText, "霍尔木兹") || strings.Contains(fullText, "海峡") {
			if strings.Contains(fullText, "封锁") || strings.Contains(fullText, "关闭") {
				war.HormuzStatus = "🚫 据报已封锁"
				war.HormuzSource = truncate(result.Title, 60)
			} else if strings.Contains(fullText, "开放") || strings.Contains(fullText, "通行") {
				war.HormuzStatus = "✅ 恢复通行"
			}
		}
		
		// 关键事件
		if war.KeyEvent == "" && len(result.Title) > 10 {
			war.KeyEvent = truncate(result.Title, 50)
		}
	}
	
	return war
}

// analyzeFinance 分析财经信息
func analyzeFinance(results []SearchResult) Finance {
	finance := Finance{
		OilPrice:   "📈 预计暴涨",
		ChinaStock: "周一预计下跌",
		HKStock:    "受冲击更大",
		SafeAssets: "黄金上涨",
	}
	
	for _, result := range results {
		fullText := result.Title + " " + result.Snippet
		
		// 石油价格
		if strings.Contains(fullText, "石油") || strings.Contains(fullText, "原油") || strings.Contains(fullText, "油价") {
			if strings.Contains(fullText, "暴涨") || strings.Contains(fullText, "飙升") || strings.Contains(fullText, "大涨") {
				finance.OilPrice = "🚀 据报暴涨"
				finance.OilSource = truncate(result.Title, 60)
			} else if strings.Contains(fullText, "下跌") || strings.Contains(fullText, "回落") || strings.Contains(fullText, "跌") {
				finance.OilPrice = "📉 据报下跌"
				finance.OilSource = truncate(result.Title, 60)
			} else if strings.Contains(fullText, "涨") {
				finance.OilPrice = "📈 据报上涨"
				finance.OilSource = truncate(result.Title, 60)
			}
		}
		
		// 关键财经事件
		if finance.KeyFinance == "" && len(result.Title) > 10 {
			finance.KeyFinance = truncate(result.Title, 50)
		}
	}
	
	return finance
}

// saveBriefing 保存简报到文件
func saveBriefing(briefing *Briefing) {
	// 创建briefings目录
	briefingsDir := "/home/zhangyufeng/.openclaw/workspace/briefings"
	if err := os.MkdirAll(briefingsDir, 0755); err != nil {
		log.Printf("⚠️ 创建目录失败: %v", err)
		return
	}
	
	// 生成文件名
	filename := fmt.Sprintf("briefing-go-%s-%d.txt", 
		time.Now().Format("2006-01-02"),
		time.Now().Hour())
	
	filepath := fmt.Sprintf("%s/%s", briefingsDir, filename)
	
	// 保存为文本
	content := formatBriefingText(briefing)
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		log.Printf("⚠️ 保存简报失败: %v", err)
	} else {
		log.Printf("简报已保存: %s", filepath)
	}
	
	// 同时保存为JSON
	jsonFile := strings.Replace(filepath, ".txt", ".json", 1)
	if data, err := json.MarshalIndent(briefing, "", "  "); err == nil {
		os.WriteFile(jsonFile, data, 0644)
	}
}

// formatBriefingText 格式化简报为文本
func formatBriefingText(briefing *Briefing) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("%s\n\n", briefing.Title))
	sb.WriteString(fmt.Sprintf("*时间*: %s (北京时间)\n", briefing.Time))
	sb.WriteString(fmt.Sprintf("*时段*: %s\n", briefing.Period))
	sb.WriteString(fmt.Sprintf("*生成*: %s\n\n", briefing.Generated))
	
	// 战争动态
	sb.WriteString("⚔️ *战争动态*\n")
	sb.WriteString(fmt.Sprintf("• 哈梅内伊: %s\n", briefing.War.HamaneyStatus))
	if briefing.War.HamaneySource != "" {
		sb.WriteString(fmt.Sprintf("  来源: %s...\n", briefing.War.HamaneySource))
	}
	sb.WriteString(fmt.Sprintf("• 霍尔木兹海峡: %s\n", briefing.War.HormuzStatus))
	if briefing.War.HormuzSource != "" {
		sb.WriteString(fmt.Sprintf("  来源: %s...\n", briefing.War.HormuzSource))
	}
	if briefing.War.KeyEvent != "" {
		sb.WriteString(fmt.Sprintf("• %s\n", briefing.War.KeyEvent))
	}
	sb.WriteString("\n")
	
	// 财经影响
	sb.WriteString("💰 *财经影响*\n")
	sb.WriteString(fmt.Sprintf("• 石油价格: %s\n", briefing.Finance.OilPrice))
	if briefing.Finance.OilSource != "" {
		sb.WriteString(fmt.Sprintf("  来源: %s...\n", briefing.Finance.OilSource))
	}
	sb.WriteString(fmt.Sprintf("• 中国股市: %s\n", briefing.Finance.ChinaStock))
	sb.WriteString(fmt.Sprintf("• 港股: %s\n", briefing.Finance.HKStock))
	sb.WriteString(fmt.Sprintf("• 避险资产: %s\n", briefing.Finance.SafeAssets))
	if briefing.Finance.KeyFinance != "" {
		sb.WriteString(fmt.Sprintf("• %s\n", briefing.Finance.KeyFinance))
	}
	sb.WriteString("\n")
	
	// 投资策略
	sb.WriteString("🎯 *投资策略*\n")
	for i, strategy := range briefing.Strategy {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, strategy))
	}
	sb.WriteString("\n")
	
	// 市场评估
	sb.WriteString("📈 *市场评估*\n")
	sb.WriteString(fmt.Sprintf("🔥 %s | 📉 %s | 🛢️ %s\n", 
		briefing.Assessment.Risk, 
		briefing.Assessment.Stock, 
		briefing.Assessment.Oil))
	sb.WriteString(fmt.Sprintf("⚠️ 建议: %s\n\n", briefing.Assessment.Advice))
	
	// 下一简报
	sb.WriteString(fmt.Sprintf("⏰ *下一简报*: %s\n", briefing.NextBrief))
	
	return sb.String()
}

// printBriefing 打印简报
func printBriefing(briefing *Briefing) {
	fmt.Println(formatBriefingText(briefing))
}

// getBriefingNumber 获取简报编号
func getBriefingNumber() int {
	// 简单实现：基于时间计算
	now := time.Now()
	return (now.Hour()/6 + 1) % 4
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}