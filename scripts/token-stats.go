// token-stats.go
// Token 使用情况统计脚本（Go版本）
// 统计 @zyf_weekly_report_bot 的 token 使用情况和费用

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SessionInfo 会话信息结构体
type SessionInfo struct {
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	CacheRead    int    `json:"cacheRead"`
	TotalTokens  int    `json:"totalTokens"`
	UpdatedAt    int64  `json:"updatedAt"`
	Model        string `json:"model"`
}

// SessionsData 会话数据
type SessionsData map[string]SessionInfo

// TokenStats 统计结果
type TokenStats struct {
	Timestamp string
	Sessions  SessionsData
	Totals    struct {
		InputTokens  int
		OutputTokens int
		CacheRead    int
		TotalTokens  int
	}
	EstimatedCost struct {
		USD float64
		CNY float64
	}
}

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 读取会话文件
func readSessionsFile() (SessionsData, error) {
	sessionsPath := "/home/zhangyufeng/.openclaw/agents/weekly_report_helper/sessions/sessions.json"
	
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("会话文件不存在: %s", sessionsPath)
	}
	
	data, err := ioutil.ReadFile(sessionsPath)
	if err != nil {
		return nil, fmt.Errorf("读取会话文件失败: %v", err)
	}
	
	var sessionsData map[string]interface{}
	if err := json.Unmarshal(data, &sessionsData); err != nil {
		return nil, fmt.Errorf("解析会话JSON失败: %v", err)
	}
	
	sessions := make(SessionsData)
	
	for sessionKey, sessionInfo := range sessionsData {
		sessionMap, ok := sessionInfo.(map[string]interface{})
		if !ok {
			continue
		}
		
		// 提取token信息
		inputTokens, _ := sessionMap["inputTokens"].(float64)
		outputTokens, _ := sessionMap["outputTokens"].(float64)
		cacheRead, _ := sessionMap["cacheRead"].(float64)
		totalTokens, _ := sessionMap["totalTokens"].(float64)
		updatedAt, _ := sessionMap["updatedAt"].(float64)
		model, _ := sessionMap["model"].(string)
		
		if inputTokens > 0 || outputTokens > 0 {
			sessions[sessionKey] = SessionInfo{
				InputTokens:  int(inputTokens),
				OutputTokens: int(outputTokens),
				CacheRead:    int(cacheRead),
				TotalTokens:  int(totalTokens),
				UpdatedAt:    int64(updatedAt),
				Model:        model,
			}
		}
	}
	
	return sessions, nil
}

// 计算统计
func calculateStats(sessions SessionsData) TokenStats {
	var stats TokenStats
	stats.Timestamp = getBeijingTime().Format("2006-01-02 15:04:05")
	stats.Sessions = sessions
	
	// 计算总计
	for _, session := range sessions {
		stats.Totals.InputTokens += session.InputTokens
		stats.Totals.OutputTokens += session.OutputTokens
		stats.Totals.CacheRead += session.CacheRead
		stats.Totals.TotalTokens += session.TotalTokens
	}
	
	// 计算估算费用
	// DeepSeek 定价:
	// 输入: $0.14/百万 tokens
	// 输出: $0.28/百万 tokens
	// 缓存读取: $0.02/百万 tokens
	// 汇率: 1 USD = 7.2 CNY
	
	inputCost := (float64(stats.Totals.InputTokens) / 1000000) * 0.14
	outputCost := (float64(stats.Totals.OutputTokens) / 1000000) * 0.28
	cacheCost := (float64(stats.Totals.CacheRead) / 1000000) * 0.02
	
	totalCostUSD := inputCost + outputCost + cacheCost
	totalCostCNY := totalCostUSD * 7.2
	
	stats.EstimatedCost.USD = totalCostUSD
	stats.EstimatedCost.CNY = totalCostCNY
	
	return stats
}

// 生成文本报告
func generateTextReport(stats TokenStats) string {
	var report strings.Builder
	
	beijingTime := getBeijingTime()
	timeStr := beijingTime.Format("2006年1月2日 15:04")
	
	report.WriteString(fmt.Sprintf("📊 @zyf_weekly_report_bot Token 使用情况统计\n"))
	report.WriteString(fmt.Sprintf("📅 统计时间: %s\n\n", timeStr))
	
	report.WriteString(fmt.Sprintf("📈 汇总统计:\n"))
	report.WriteString(fmt.Sprintf("├─ 会话数量: %d\n", len(stats.Sessions)))
	report.WriteString(fmt.Sprintf("├─ 输入 Token: %s\n", formatNumber(stats.Totals.InputTokens)))
	report.WriteString(fmt.Sprintf("├─ 输出 Token: %s\n", formatNumber(stats.Totals.OutputTokens)))
	report.WriteString(fmt.Sprintf("├─ 缓存读取: %s\n", formatNumber(stats.Totals.CacheRead)))
	report.WriteString(fmt.Sprintf("├─ 总 Token: %s\n", formatNumber(stats.Totals.TotalTokens)))
	report.WriteString(fmt.Sprintf("├─ 估算费用: $%.6f\n", stats.EstimatedCost.USD))
	report.WriteString(fmt.Sprintf("└─ 约 %.4f 元\n\n", stats.EstimatedCost.CNY))
	
	// 会话详情
	if len(stats.Sessions) > 0 {
		report.WriteString(fmt.Sprintf("📋 会话详情:\n"))
		index := 1
		for sessionKey, session := range stats.Sessions {
			shortKey := sessionKey
			if len(sessionKey) > 40 {
				shortKey = sessionKey[:40] + "..."
			}
			
			updatedAt := time.Unix(session.UpdatedAt/1000, 0)
			updatedStr := updatedAt.In(time.FixedZone("CST", 8*3600)).Format("2006/1/2 15:04:05")
			
			report.WriteString(fmt.Sprintf("%d. %s\n", index, shortKey))
			report.WriteString(fmt.Sprintf("   ├─ 输入: %s\n", formatNumber(session.InputTokens)))
			report.WriteString(fmt.Sprintf("   ├─ 输出: %s\n", formatNumber(session.OutputTokens)))
			report.WriteString(fmt.Sprintf("   ├─ 缓存: %s\n", formatNumber(session.CacheRead)))
			report.WriteString(fmt.Sprintf("   ├─ 总计: %s\n", formatNumber(session.TotalTokens)))
			report.WriteString(fmt.Sprintf("   ├─ 模型: %s\n", session.Model))
			report.WriteString(fmt.Sprintf("   └─ 更新时间: %s\n", updatedStr))
			index++
		}
		report.WriteString("\n")
	}
	
	report.WriteString(fmt.Sprintf("💡 说明:\n"))
	report.WriteString(fmt.Sprintf("- 基于 DeepSeek 标准定价估算\n"))
	report.WriteString(fmt.Sprintf("- 输入: $0.14/百万 tokens\n"))
	report.WriteString(fmt.Sprintf("- 输出: $0.28/百万 tokens\n"))
	report.WriteString(fmt.Sprintf("- 缓存读取: $0.02/百万 tokens\n"))
	report.WriteString(fmt.Sprintf("- 汇率: 1 USD = 7.2 CNY\n"))
	
	return report.String()
}

// 生成JSON报告
func generateJSONReport(stats TokenStats) map[string]interface{} {
	report := map[string]interface{}{
		"timestamp": stats.Timestamp,
		"summary": map[string]interface{}{
			"totalSessions":      len(stats.Sessions),
			"totalInputTokens":   stats.Totals.InputTokens,
			"totalOutputTokens":  stats.Totals.OutputTokens,
			"totalCacheRead":     stats.Totals.CacheRead,
			"totalTokens":        stats.Totals.TotalTokens,
			"estimatedCostUSD":   fmt.Sprintf("%.6f", stats.EstimatedCost.USD),
			"estimatedCostCNY":   fmt.Sprintf("%.4f", stats.EstimatedCost.CNY),
		},
		"sessions": stats.Sessions,
		"details": map[string]interface{}{
			"inputTokens":  stats.Totals.InputTokens,
			"outputTokens": stats.Totals.OutputTokens,
			"cacheRead":    stats.Totals.CacheRead,
			"totalTokens":  stats.Totals.TotalTokens,
			"estimatedCost": map[string]interface{}{
				"usd": stats.EstimatedCost.USD,
				"cny": stats.EstimatedCost.CNY,
			},
		},
	}
	
	return report
}

// 保存报告
func saveReports(stats TokenStats, jsonReport map[string]interface{}, textReport string) error {
	// 创建报告目录
	reportsDir := "/home/zhangyufeng/.openclaw/workspace/token-reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("创建报告目录失败: %v", err)
	}
	
	dateStr := getBeijingTime().Format("2006-01-02")
	
	// 保存JSON报告
	jsonFile := filepath.Join(reportsDir, fmt.Sprintf("token-stats-%s.json", dateStr))
	
	// 读取现有报告（如果存在）
	var dailyReports []map[string]interface{}
	if data, err := ioutil.ReadFile(jsonFile); err == nil {
		if err := json.Unmarshal(data, &dailyReports); err != nil {
			dailyReports = []map[string]interface{}{}
		}
	}
	
	// 添加新报告
	dailyReports = append(dailyReports, jsonReport)
	
	// 只保留最近30天的报告
	if len(dailyReports) > 30 {
		dailyReports = dailyReports[len(dailyReports)-30:]
	}
	
	// 保存JSON
	jsonData, err := json.MarshalIndent(dailyReports, "", "  ")
	if err != nil {
		return fmt.Errorf("编码JSON失败: %v", err)
	}
	
	if err := ioutil.WriteFile(jsonFile, jsonData, 0644); err != nil {
		return fmt.Errorf("保存JSON报告失败: %v", err)
	}
	
	// 保存文本报告
	textFile := filepath.Join(reportsDir, fmt.Sprintf("token-stats-%s.txt", dateStr))
	if err := ioutil.WriteFile(textFile, []byte(textReport), 0644); err != nil {
		return fmt.Errorf("保存文本报告失败: %v", err)
	}
	
	log.Printf("报告已保存: %s", jsonFile)
	log.Printf("文本报告已保存: %s", textFile)
	
	return nil
}

// 格式化数字（添加千位分隔符）
func formatNumber(n int) string {
	str := strconv.Itoa(n)
	var result strings.Builder
	
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	
	return result.String()
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	log.Println("开始统计 token 使用情况...")
	
	// 读取会话数据
	sessions, err := readSessionsFile()
	if err != nil {
		log.Fatalf("读取会话数据失败: %v", err)
	}
	
	if len(sessions) == 0 {
		log.Println("警告: 未找到有效的会话数据")
	}
	
	// 计算统计
	stats := calculateStats(sessions)
	
	// 生成报告
	textReport := generateTextReport(stats)
	jsonReport := generateJSONReport(stats)
	
	// 保存报告
	if err := saveReports(stats, jsonReport, textReport); err != nil {
		log.Printf("保存报告失败: %v", err)
	}
	
	// 输出到控制台
	fmt.Println(textReport)
	
	log.Println("Token 统计完成")
}