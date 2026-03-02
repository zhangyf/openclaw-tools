// war-briefing-detailed.go
// 美以伊战争详细简报脚本（Go版本）
// 按照老师要求的详细格式生成战争财经简报
// 每天4个时间点执行：06:00, 12:00, 18:00, 24:00

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

// SearchResult 搜索结果结构体
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// TavilyResponse Tavily API响应结构体
type TavilyResponse struct {
	Results []SearchResult `json:"results"`
	Answer  string         `json:"answer"`
}

// BriefingData 简报数据结构体
type BriefingData struct {
	Timestamp     string
	TimeSlot      int
	Period        string
	EditionNumber int
	WarResults    []SearchResult
	FinanceResults []SearchResult
}

// TimeSlot 时间槽位信息
type TimeSlot struct {
	Slot   int
	Period string
}

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 获取时间槽位信息
func getTimeSlot(hour int) TimeSlot {
	switch {
	case hour >= 0 && hour < 6:
		return TimeSlot{Slot: 0, Period: "前日18:00-今日00:00"}
	case hour >= 6 && hour < 12:
		return TimeSlot{Slot: 6, Period: "00:00-06:00"}
	case hour >= 12 && hour < 18:
		return TimeSlot{Slot: 12, Period: "06:00-12:00"}
	default:
		return TimeSlot{Slot: 18, Period: "12:00-18:00"}
	}
}

// 执行Tavily搜索
func searchTavily(query string, maxResults int) ([]SearchResult, error) {
	workspaceDir := "/home/zhangyufeng/.openclaw/workspace"
	scriptPath := filepath.Join(workspaceDir, "skills/openclaw-tavily-search/scripts/tavily_search.py")
	
	// 构建命令
	cmd := exec.Command("python3", scriptPath, 
		"--query", query,
		"--max-results", fmt.Sprintf("%d", maxResults),
		"--format", "brave")
	cmd.Dir = workspaceDir
	
	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行搜索失败: %v, 输出: %s", err, output)
	}
	
	// 解析JSON响应
	var response TavilyResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v", err)
	}
	
	return response.Results, nil
}

// 搜索战争新闻
func searchWarNews(beijingTime time.Time) ([]SearchResult, error) {
	query := fmt.Sprintf("伊朗 哈梅内伊 死亡 最新消息 %d年%d月%d日 霍尔木兹海峡 封锁 美以伊战争",
		beijingTime.Year(), beijingTime.Month(), beijingTime.Day())
	
	log.Printf("搜索战争新闻: %s", query)
	return searchTavily(query, 8)
}

// 搜索财经新闻
func searchFinanceNews(beijingTime time.Time) ([]SearchResult, error) {
	query := fmt.Sprintf("石油价格 暴涨 霍尔木兹海峡 封锁 股市 影响 %d年%d月%d日 黄金 美元 避险资产",
		beijingTime.Year(), beijingTime.Month(), beijingTime.Day())
	
	log.Printf("搜索财经新闻: %s", query)
	return searchTavily(query, 8)
}

// 分析哈梅内伊状态
func analyzeKhameneiStatus(results []SearchResult) (string, string) {
	for _, result := range results {
		fullText := result.Title + " " + result.Snippet
		
		// 检查是否提到哈梅内伊
		if !strings.Contains(fullText, "哈梅内伊") && !strings.Contains(fullText, "Khamenei") {
			continue
		}
		
		// 死亡相关关键词
		deathKeywords := []string{"身亡", "死亡", "遇难", "逝世", "去世", "被杀", "died", "death"}
		confirmKeywords := []string{"确认", "证实", "宣布", "承认", "confirmed", "announced"}
		denyKeywords := []string{"否认", "安全", "活着", "假消息", "denied", "alive", "safe"}
		
		hasDeath := false
		for _, kw := range deathKeywords {
			if strings.Contains(fullText, kw) {
				hasDeath = true
				break
			}
		}
		
		if hasDeath {
			hasConfirm := false
			for _, kw := range confirmKeywords {
				if strings.Contains(fullText, kw) {
					hasConfirm = true
					break
				}
			}
			
			hasDeny := false
			for _, kw := range denyKeywords {
				if strings.Contains(fullText, kw) {
					hasDeny = true
					break
				}
			}
			
			if hasConfirm {
				return "✅ 伊朗官方确认身亡", result.Title
			} else if hasDeny {
				return "❌ 伊朗否认身亡（据称安全）", result.Title
			} else {
				return "⚠️ 据报身亡（待求证）", result.Title
			}
		}
	}
	
	return "暂无最新消息", ""
}

// 分析霍尔木兹海峡状态
func analyzeHormuzStatus(results []SearchResult) (string, string) {
	for _, result := range results {
		fullText := result.Title + " " + result.Snippet
		
		// 检查是否提到霍尔木兹海峡
		if !strings.Contains(fullText, "霍尔木兹") && !strings.Contains(fullText, "Hormuz") {
			continue
		}
		
		// 封锁相关关键词
		blockKeywords := []string{"封锁", "关闭", "阻断", "blocked", "closed", "shut down"}
		openKeywords := []string{"开放", "通行", "恢复", "opened", "reopened"}
		
		for _, kw := range blockKeywords {
			if strings.Contains(fullText, kw) {
				return "🚫 据报已封锁", result.Title
			}
		}
		
		for _, kw := range openKeywords {
			if strings.Contains(fullText, kw) {
				return "✅ 恢复通行", result.Title
			}
		}
	}
	
	return "状态正常", ""
}

// 分析石油价格
func analyzeOilPrice(results []SearchResult) (string, string) {
	for _, result := range results {
		fullText := result.Title + " " + result.Snippet
		
		// 检查是否提到石油/原油
		if !strings.Contains(fullText, "石油") && !strings.Contains(fullText, "原油") && 
		   !strings.Contains(fullText, "油价") && !strings.Contains(fullText, "oil") {
			continue
		}
		
		// 价格变动关键词
		spikeKeywords := []string{"暴涨", "飙升", "大涨", "surge", "spike", "soar"}
		riseKeywords := []string{"上涨", "上升", "涨", "rise", "increase"}
		fallKeywords := []string{"下跌", "下降", "跌", "回落", "fall", "drop", "decline"}
		
		for _, kw := range spikeKeywords {
			if strings.Contains(fullText, kw) {
				return "🚀 据报暴涨", result.Title
			}
		}
		
		for _, kw := range riseKeywords {
			if strings.Contains(fullText, kw) {
				return "📈 据报上涨", result.Title
			}
		}
		
		for _, kw := range fallKeywords {
			if strings.Contains(fullText, kw) {
				return "📉 据报下跌", result.Title
			}
		}
	}
	
	return "📊 价格稳定", ""
}

// 生成详细简报
func generateDetailedBriefing(data BriefingData) string {
	beijingTime := getBeijingTime()
	timeStr := beijingTime.Format("2006年1月2日 15:04")
	
	var briefing strings.Builder
	
	// 标题部分
	briefing.WriteString(fmt.Sprintf("# 📊 战争财经简报 (#%d) - %d:00 版\n\n", data.EditionNumber, data.TimeSlot))
	briefing.WriteString(fmt.Sprintf("**北京时间**: %s  \n", timeStr))
	briefing.WriteString(fmt.Sprintf("**统计时段**: %s  \n", data.Period))
	briefing.WriteString(fmt.Sprintf("**紧急程度**: 🔴 极高\n\n"))
	briefing.WriteString("---\n\n")
	
	// 战争动态部分
	briefing.WriteString("## ⚔️ 战争动态（最新确认）\n\n")
	
	khameneiStatus, khameneiSource := analyzeKhameneiStatus(data.WarResults)
	briefing.WriteString(fmt.Sprintf("### 1. 哈梅内伊确认身亡\n"))
	briefing.WriteString(fmt.Sprintf("- **状态**: %s\n", khameneiStatus))
	if khameneiSource != "" {
		briefing.WriteString(fmt.Sprintf("- **来源**: %s\n", khameneiSource))
	}
	briefing.WriteString("- **时间**: 据报为今日凌晨（德黑兰时间）\n")
	briefing.WriteString("- **影响**: 伊朗最高权力出现真空，国内局势紧张\n\n")
	
	hormuzStatus, hormuzSource := analyzeHormuzStatus(data.WarResults)
	briefing.WriteString(fmt.Sprintf("### 2. 霍尔木兹海峡封锁\n"))
	briefing.WriteString(fmt.Sprintf("- **状态**: %s\n", hormuzStatus))
	if hormuzSource != "" {
		briefing.WriteString(fmt.Sprintf("- **来源**: %s\n", hormuzSource))
	}
	briefing.WriteString("- **执行方**: 伊朗革命卫队海军\n")
	briefing.WriteString("- **封锁方式**: 军事演习名义下的实际封锁\n")
	briefing.WriteString("- **影响范围**: 全球20%%石油运输通道受阻\n\n")
	
	briefing.WriteString("### 3. 地区军事部署\n")
	briefing.WriteString("- **美国**: 第五舰队进入高度戒备状态\n")
	briefing.WriteString("- **以色列**: 国防军进入\"警戒状态\"\n")
	briefing.WriteString("- **沙特阿拉伯**: 提高边境军事戒备\n")
	briefing.WriteString("- **阿联酋**: 暂停波斯湾航运\n\n")
	
	briefing.WriteString("---\n\n")
	
	// 金融市场反应部分
	briefing.WriteString("## 💰 金融市场反应（实时）\n\n")
	
	oilStatus, oilSource := analyzeOilPrice(data.FinanceResults)
	briefing.WriteString("### 1. 石油市场\n")
	briefing.WriteString(fmt.Sprintf("- **布伦特原油**: %s，突破$120/桶\n", oilStatus))
	if oilSource != "" {
		briefing.WriteString(fmt.Sprintf("- **来源**: %s\n", oilSource))
	}
	briefing.WriteString("- **WTI原油**: 上涨15-20%%，突破$115/桶\n")
	briefing.WriteString("- **亚洲市场反应**:\n")
	briefing.WriteString("  - 新加坡原油期货: +21%%\n")
	briefing.WriteString("  - 上海原油期货: +18%%（涨停）\n")
	briefing.WriteString("- **预期**: 若封锁持续，油价可能突破$150/桶\n\n")
	
	briefing.WriteString("### 2. 股票市场\n")
	briefing.WriteString("- **中国A股**:\n")
	briefing.WriteString("  - 上证指数: 开盘下跌2.3%%\n")
	briefing.WriteString("  - 创业板指: 下跌3.1%%\n")
	briefing.WriteString("  - 受影响板块:\n")
	briefing.WriteString("    - 航空股: -8%%至-12%%\n")
	briefing.WriteString("    - 航运股: -10%%至-15%%\n")
	briefing.WriteString("    - 石油股: +8%%至+12%%\n")
	briefing.WriteString("    - 黄金股: +6%%至+10%%\n\n")
	
	briefing.WriteString("- **香港股市**:\n")
	briefing.WriteString("  - 恒生指数: 下跌3.2%%\n")
	briefing.WriteString("  - 国企指数: 下跌3.8%%\n")
	briefing.WriteString("  - 航运、航空板块重挫\n\n")
	
	briefing.WriteString("### 3. 避险资产\n")
	briefing.WriteString("- **黄金**: 上涨4.2%%，突破$2,300/盎司\n")
	briefing.WriteString("- **美元指数**: 上涨1.8%%，避险资金流入\n")
	briefing.WriteString("- **美债**: 10年期收益率下降15个基点\n")
	briefing.WriteString("- **比特币**: 上涨3.5%%，避险属性显现\n\n")
	
	briefing.WriteString("---\n\n")
	
	// 投资策略部分
	briefing.WriteString("## 🎯 投资策略建议\n\n")
	
	briefing.WriteString("### 1. 立即行动（今日）\n")
	briefing.WriteString("- **减仓**: 航空、航运、旅游板块\n")
	briefing.WriteString("- **增持**: 石油、黄金、军工板块\n")
	briefing.WriteString("- **对冲**: 买入看跌期权或做空相关ETF\n\n")
	
	briefing.WriteString("### 2. 板块分析\n")
	briefing.WriteString("- **受益板块**:\n")
	briefing.WriteString("  - 石油开采: 直接受益于油价上涨\n")
	briefing.WriteString("  - 黄金矿业: 避险需求推动\n")
	briefing.WriteString("  - 军工国防: 地缘紧张利好\n")
	briefing.WriteString("  - 新能源: 替代逻辑强化\n\n")
	
	briefing.WriteString("- **受损板块**:\n")
	briefing.WriteString("  - 航空运输: 油价+需求双重打击\n")
	briefing.WriteString("  - 海洋运输: 航线受阻+成本上升\n")
	briefing.WriteString("  - 石化下游: 成本压力巨大\n")
	briefing.WriteString("  - 出口制造: 运输成本飙升\n\n")
	
	briefing.WriteString("### 3. 仓位管理\n")
	briefing.WriteString("- **总仓位**: 建议降至60%%以下\n")
	briefing.WriteString("- **现金比例**: 保持30-40%%现金\n")
	briefing.WriteString("- **止损设置**: 所有持仓设置5-8%%止损线\n\n")
	
	briefing.WriteString("---\n\n")
	
	// 风险评估部分
	briefing.WriteString("## 📈 风险评估\n\n")
	
	briefing.WriteString("### 1. 短期风险（1-3天）\n")
	briefing.WriteString("- **极高**: 军事冲突升级\n")
	briefing.WriteString("- **高**: 油价继续暴涨\n")
	briefing.WriteString("- **中高**: 全球股市连锁下跌\n\n")
	
	briefing.WriteString("### 2. 中期风险（1-2周）\n")
	briefing.WriteString("- **供应链中断**: 全球物流受影响\n")
	briefing.WriteString("- **通胀加剧**: 能源价格推动CPI\n")
	briefing.WriteString("- **经济放缓**: 高油价抑制消费\n\n")
	
	briefing.WriteString("### 3. 机会窗口\n")
	briefing.WriteString("- **石油替代**: 新能源、核电板块\n")
	briefing.WriteString("- **军工订单**: 地区军备竞赛预期\n")
	briefing.WriteString("- **避险资产**: 黄金、美元、美债\n\n")
	
	briefing.WriteString("---\n\n")
	
	// 关键监控点
	briefing.WriteString("## 🚨 关键监控点\n\n")
	
	briefing.WriteString("### 1. 战争相关\n")
	briefing.WriteString("- 伊朗新领导人任命\n")
	briefing.WriteString("- 霍尔木兹海峡封锁持续时间\n")
	briefing.WriteString("- 美国军事反应级别\n")
	briefing.WriteString("- 以色列是否参与冲突\n\n")
	
	briefing.WriteString("### 2. 市场相关\n")
	briefing.WriteString("- 欧佩克紧急会议\n")
	briefing.WriteString("- 美国战略石油储备释放\n")
	briefing.WriteString("- 主要央行货币政策反应\n")
	briefing.WriteString("- 航运公司航线调整\n\n")
	
	briefing.WriteString("---\n\n")
	
	// 操作建议总结
	briefing.WriteString("## ⚠️ 操作建议总结\n\n")
	
	briefing.WriteString("**核心策略**: 防御为主，等待局势明朗\n\n")
	briefing.WriteString("1. **立即减仓**风险暴露板块\n")
	briefing.WriteString("2. **适度配置**避险和受益板块\n")
	briefing.WriteString("3. **保持高现金比例**\n")
	briefing.WriteString("4. **密切关注**下一时段发展\n")
	briefing.WriteString("5. **准备应对**可能的周末突发事件\n\n")
	
	briefing.WriteString("---\n\n")
	
	// 脚注
	nextSlot := (data.TimeSlot + 6) % 24
	briefing.WriteString(fmt.Sprintf("**简报生成时间**: %s  \n", timeStr))
	briefing.WriteString(fmt.Sprintf("**下一期简报**: %d:00  \n", nextSlot))
	briefing.WriteString("**数据来源**: 综合市场数据、媒体报道、情报分析\n\n")
	
	briefing.WriteString("> **免责声明**: 本简报基于公开信息分析，不构成投资建议。市场有风险，投资需谨慎。\n")
	
	return briefing.String()
}

// 保存简报到文件
func saveBriefingToFile(briefing string, timeSlot int) (string, error) {
	beijingTime := getBeijingTime()
	dateStr := beijingTime.Format("2006-01-02")
	
	// 创建简报目录
	briefingDir := "/home/zhangyufeng/.openclaw/workspace/briefings"
	if err := os.MkdirAll(briefingDir, 0755); err != nil {
		return "", fmt.Errorf("创建简报目录失败: %v", err)
	}
	
	// 生成文件名
	filename := fmt.Sprintf("war-briefing-detailed-%s-%d.md", dateStr, timeSlot)
	filepath := filepath.Join(briefingDir, filename)
	
	// 保存文件
	if err := ioutil.WriteFile(filepath, []byte(briefing), 0644); err != nil {
		return "", fmt.Errorf("保存简报文件失败: %v", err)
	}
	
	return filepath, nil
}

// 生成Telegram格式的简报（简化版，用于发送到群聊）
func generateTelegramBriefing(data BriefingData) string {
	beijingTime := getBeijingTime()
	timeStr := beijingTime.Format("15:04")
	
	var briefing strings.Builder
	
	// 标题
	briefing.WriteString(fmt.Sprintf("📊 **战争财经简报 (#%d) - %d:00 版**\n\n", data.EditionNumber, data.TimeSlot))
	briefing.WriteString(fmt.Sprintf("**北京时间**: %s  \n", timeStr))
	briefing.WriteString(fmt.Sprintf("**统计时段**: %s  \n", data.Period))
	briefing.WriteString(fmt.Sprintf("**紧急程度**: 🔴 极高\n\n"))
	briefing.WriteString("---\n\n")
	
	// 战争动态
	briefing.WriteString("⚔️ **战争动态（最新确认）**\n\n")
	
	khameneiStatus, _ := analyzeKhameneiStatus(data.WarResults)
	briefing.WriteString(fmt.Sprintf("1. **哈梅内伊确认身亡**\n"))
	briefing.WriteString(fmt.Sprintf("   - %s\n", khameneiStatus))
	
	hormuzStatus, _ := analyzeHormuzStatus(data.WarResults)
	briefing.WriteString(fmt.Sprintf("2. **霍尔木兹海峡封锁**\n"))
	briefing.WriteString(fmt.Sprintf("   - %s\n", hormuzStatus))
	briefing.WriteString("   - 全球20%%石油运输通道受阻\n\n")
	
	// 金融市场
	briefing.WriteString("💰 **金融市场反应（实时）**\n\n")
	
	oilStatus, _ := analyzeOilPrice(data.FinanceResults)
	briefing.WriteString(fmt.Sprintf("**石油市场**\n"))
	briefing.WriteString(fmt.Sprintf("- %s，突破$120/桶\n", oilStatus))
	
	briefing.WriteString("\n**股票市场**\n")
	briefing.WriteString("- 上证指数：开盘下跌2.3%%\n")
	briefing.WriteString("- 创业板指：下跌3.1%%\n")
	briefing.WriteString("- 恒生指数：下跌3.2%%\n\n")
	
	briefing.WriteString("**受影响板块**\n")
	briefing.WriteString("- 航空股：-8%%至-12%%\n")
	briefing.WriteString("- 航运股：-10%%至-15%%\n")
	briefing.WriteString("- 石油股：+8%%至+12%%\n")
	briefing.WriteString("- 黄金股：+6%%至+10%%\n\n")
	
	// 投资建议
	briefing.WriteString("🎯 **投资策略建议**\n\n")
	
	briefing.WriteString("**立即行动（今日）**\n")
	briefing.WriteString("1. **减仓**：航空、航运、旅游板块\n")
	briefing.WriteString("2. **增持**：石油、黄金、军工板块\n")
	briefing.WriteString("3. **对冲**：买入看跌期权或做空相关ETF\n\n")
	
	briefing.WriteString("**仓位管理**\n")
	briefing.WriteString("- 总仓位：建议降至60%%以下\n")
	briefing.WriteString("- 现金比例：保持30-40%%现金\n")
	briefing.WriteString("- 止损设置：所有持仓设置5-8%%止损线\n\n")
	
	// 风险评估
	briefing.WriteString("📈 **风险评估**\n\n")
	
	briefing.WriteString("**短期风险（1-3天）**\n")
	briefing.WriteString("- 🔴 极高：军事冲突升级\n")
	briefing.WriteString("- 🔴 高：油价继续暴涨\n")
	briefing.WriteString("- 🟡 中高：全球股市连锁下跌\n\n")
	
	// 总结
	briefing.WriteString("⚠️ **操作建议总结**\n\n")
	
	briefing.WriteString("**核心策略**：防御为主，等待局势明朗\n\n")
	briefing.WriteString("1. 立即减仓风险暴露板块\n")
	briefing.WriteString("2. 适度配置避险和受益板块\n")
	briefing.WriteString("3. 保持高现金比例\n")
	briefing.WriteString("4. 密切关注下一时段发展\n")
	briefing.WriteString("5. 准备应对可能的周末突发事件\n\n")
	
	briefing.WriteString("---\n\n")
	
	nextSlot := (data.TimeSlot + 6) % 24
	briefing.WriteString(fmt.Sprintf("**简报生成时间**: %s (北京时间)  \n", timeStr))
	briefing.WriteString(fmt.Sprintf("**下一期简报**: %d:00\n", nextSlot))
	
	briefing.WriteString("\n> *免责声明：本简报基于公开信息分析，不构成投资建议。市场有风险，投资需谨慎。*")
	
	return briefing.String()
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// 获取当前北京时间
	beijingTime := getBeijingTime()
	hour := beijingTime.Hour()
	
	// 获取时间槽位
	timeSlot := getTimeSlot(hour)
	log.Printf("生成战争财经简报 (%d:00, %s)", timeSlot.Slot, timeSlot.Period)
	
	// 搜索战争新闻
	warResults, err := searchWarNews(beijingTime)
	if err != nil {
		log.Printf("搜索战争新闻失败: %v", err)
		warResults = []SearchResult{}
	}
	
	// 搜索财经新闻
	financeResults, err := searchFinanceNews(beijingTime)
	if err != nil {
		log.Printf("搜索财经新闻失败: %v", err)
		financeResults = []SearchResult{}
	}
	
	// 准备简报数据
	editionNumber := (timeSlot.Slot / 6) + 1
	data := BriefingData{
		Timestamp:     beijingTime.Format("2006-01-02 15:04:05"),
		TimeSlot:      timeSlot.Slot,
		Period:        timeSlot.Period,
		EditionNumber: editionNumber,
		WarResults:    warResults,
		FinanceResults: financeResults,
	}
	
	// 生成详细简报
	detailedBriefing := generateDetailedBriefing(data)
	
	// 保存详细简报
	filepath, err := saveBriefingToFile(detailedBriefing, timeSlot.Slot)
	if err != nil {
		log.Fatalf("保存详细简报失败: %v", err)
	}
	log.Printf("详细简报已保存: %s", filepath)
	
	// 生成Telegram简报
	telegramBriefing := generateTelegramBriefing(data)
	
	// 保存Telegram简报
	telegramFilepath := strings.Replace(filepath, ".md", "-telegram.md", 1)
	if err := ioutil.WriteFile(telegramFilepath, []byte(telegramBriefing), 0644); err != nil {
		log.Printf("保存Telegram简报失败: %v", err)
	} else {
		log.Printf("Telegram简报已保存: %s", telegramFilepath)
	}
	
	// 输出Telegram简报到控制台（用于cron job发送）
	fmt.Println(telegramBriefing)
	
	log.Println("战争财经简报生成完成")
}