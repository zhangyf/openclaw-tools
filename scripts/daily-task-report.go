// daily-task-report.go
// 每日任务汇总报告脚本（Go版本）
// 生成18:00的任务执行情况报告，支持发送到Telegram群聊

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Task 任务结构体
type Task struct {
	ID          string    `json:"id"`
	Created     time.Time `json:"created"`
	Deadline    time.Time `json:"deadline"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Details     string    `json:"details"`
	Progress    string    `json:"progress"`
	CompletedAt time.Time `json:"completedAt"`
	FailedAt    time.Time `json:"failedAt"`
	Error       string    `json:"error"`
}

// DailyStats 每日统计
type DailyStats struct {
	TotalTasks     int
	CompletedToday int
	FailedToday    int
	ActiveTasks    int
	OverdueTasks   int
	CompletionRate float64
}

// TaskDetails 任务详情
type TaskDetails struct {
	Completed []Task
	Failed    []Task
	Active    []Task
	Overdue   []Task
}

// TokenUsage Token使用情况
type TokenUsage struct {
	TotalTokens   int
	EstimatedCost float64
	Success       bool
	Error         string
}

// 配置常量
const (
	TasksDir       = "/home/zhangyufeng/.openclaw/workspace/tasks"
	ActiveDir      = TasksDir + "/active"
	CompletedDir   = TasksDir + "/completed"
	FailedDir      = TasksDir + "/failed"
	ReportsDir     = TasksDir + "/reports"
	TelegramChatID = "-5149902750" // 张府群聊ID
)

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 解析时间字符串
func parseTime(timeStr string) (time.Time, error) {
	// 尝试多种时间格式
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
		time.RFC3339Nano,
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("无法解析时间: %s", timeStr)
}

// 读取任务文件
func readTaskFile(filepath string) (Task, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return Task{}, err
	}
	
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return Task{}, err
	}
	
	// 解析时间字段
	if task.Created.IsZero() {
		if createdStr, ok := taskMap(data)["created"].(string); ok {
			if t, err := parseTime(createdStr); err == nil {
				task.Created = t
			}
		}
	}
	
	if task.Deadline.IsZero() {
		if deadlineStr, ok := taskMap(data)["deadline"].(string); ok {
			if t, err := parseTime(deadlineStr); err == nil {
				task.Deadline = t
			}
		}
	}
	
	if task.CompletedAt.IsZero() {
		if completedStr, ok := taskMap(data)["completedAt"].(string); ok {
			if t, err := parseTime(completedStr); err == nil {
				task.CompletedAt = t
			}
		}
	}
	
	if task.FailedAt.IsZero() {
		if failedStr, ok := taskMap(data)["failedAt"].(string); ok {
			if t, err := parseTime(failedStr); err == nil {
				task.FailedAt = t
			}
		}
	}
	
	return task, nil
}

// 辅助函数：将JSON解析为map
func taskMap(data []byte) map[string]interface{} {
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}

// 获取当日统计
func getDailyStats() DailyStats {
	now := getBeijingTime()
	today := now.Format("2006-01-02")
	startOfDay, _ := time.ParseInLocation("2006-01-02", today, now.Location())
	
	var stats DailyStats
	
	// 检查已完成任务
	if files, err := ioutil.ReadDir(CompletedDir); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				taskPath := filepath.Join(CompletedDir, file.Name())
				task, err := readTaskFile(taskPath)
				if err != nil {
					continue
				}
				
				stats.TotalTasks++
				if !task.CompletedAt.IsZero() && task.CompletedAt.After(startOfDay) {
					stats.CompletedToday++
				}
			}
		}
	}
	
	// 检查失败任务
	if files, err := ioutil.ReadDir(FailedDir); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				taskPath := filepath.Join(FailedDir, file.Name())
				task, err := readTaskFile(taskPath)
				if err != nil {
					continue
				}
				
				stats.TotalTasks++
				if !task.FailedAt.IsZero() && task.FailedAt.After(startOfDay) {
					stats.FailedToday++
				}
			}
		}
	}
	
	// 检查进行中任务
	if files, err := ioutil.ReadDir(ActiveDir); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				taskPath := filepath.Join(ActiveDir, file.Name())
				task, err := readTaskFile(taskPath)
				if err != nil {
					continue
				}
				
				stats.TotalTasks++
				stats.ActiveTasks++
				
				// 检查是否过期
				if !task.Deadline.IsZero() && task.Deadline.Before(now) {
					stats.OverdueTasks++
				}
			}
		}
	}
	
	// 计算完成率
	if stats.TotalTasks > 0 {
		stats.CompletionRate = float64(stats.CompletedToday) / float64(stats.TotalTasks) * 100
	}
	
	return stats
}

// 获取Token使用统计
func getTokenUsage() TokenUsage {
	// 使用绝对路径调用Go版本的token统计脚本
	tokenStatsPath := "/home/zhangyufeng/.openclaw/workspace/scripts/token-stats"
	
	if _, err := os.Stat(tokenStatsPath); os.IsNotExist(err) {
		return TokenUsage{
			Error: "token统计脚本不存在: " + tokenStatsPath,
		}
	}
	
	cmd := exec.Command(tokenStatsPath)
	output, err := cmd.Output()
	if err != nil {
		return TokenUsage{
			Error: fmt.Sprintf("执行token统计失败: %v", err),
		}
	}
	
	// 从输出中提取信息
	outputStr := string(output)
	var usage TokenUsage
	
	// 解析总Token数
	if idx := strings.Index(outputStr, "总 Token: "); idx != -1 {
		start := idx + len("总 Token: ")
		end := strings.Index(outputStr[start:], "\n")
		if end != -1 {
			tokenStr := strings.ReplaceAll(outputStr[start:start+end], ",", "")
			fmt.Sscanf(tokenStr, "%d", &usage.TotalTokens)
		}
	}
	
	// 解析估算成本
	if idx := strings.Index(outputStr, "约 "); idx != -1 {
		start := idx + len("约 ")
		end := strings.Index(outputStr[start:], " 元")
		if end != -1 {
			costStr := outputStr[start : start+end]
			fmt.Sscanf(costStr, "%f", &usage.EstimatedCost)
		}
	}
	
	usage.Success = usage.TotalTokens > 0 || usage.EstimatedCost > 0
	return usage
}

// 获取任务详情
func getTaskDetails() TaskDetails {
	now := getBeijingTime()
	today := now.Format("2006-01-02")
	startOfDay, _ := time.ParseInLocation("2006-01-02", today, now.Location())
	
	var details TaskDetails
	
	// 获取已完成任务
	if files, err := ioutil.ReadDir(CompletedDir); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				taskPath := filepath.Join(CompletedDir, file.Name())
				task, err := readTaskFile(taskPath)
				if err != nil {
					continue
				}
				
				if !task.CompletedAt.IsZero() && task.CompletedAt.After(startOfDay) {
					details.Completed = append(details.Completed, task)
				}
			}
		}
	}
	
	// 获取失败任务
	if files, err := ioutil.ReadDir(FailedDir); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				taskPath := filepath.Join(FailedDir, file.Name())
				task, err := readTaskFile(taskPath)
				if err != nil {
					continue
				}
				
				if !task.FailedAt.IsZero() && task.FailedAt.After(startOfDay) {
					details.Failed = append(details.Failed, task)
				}
			}
		}
	}
	
	// 获取进行中和过期任务
	if files, err := ioutil.ReadDir(ActiveDir); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				taskPath := filepath.Join(ActiveDir, file.Name())
				task, err := readTaskFile(taskPath)
				if err != nil {
					continue
				}
				
				isOverdue := !task.Deadline.IsZero() && task.Deadline.Before(now)
				if isOverdue {
					details.Overdue = append(details.Overdue, task)
				} else {
					details.Active = append(details.Active, task)
				}
			}
		}
	}
	
	// 按优先级排序
	sort.Slice(details.Active, func(i, j int) bool {
		priorityOrder := map[string]int{
			"critical": 0,
			"high":     1,
			"medium":   2,
			"low":      3,
			"normal":   4,
		}
		return priorityOrder[details.Active[i].Priority] < priorityOrder[details.Active[j].Priority]
	})
	
	return details
}

// 生成报告
func generateReport() string {
	now := getBeijingTime()
	timeStr := now.Format("2006/1/2 15:04:05")
	
	stats := getDailyStats()
	tokenUsage := getTokenUsage()
	taskDetails := getTaskDetails()
	
	var report strings.Builder
	
	report.WriteString(fmt.Sprintf("📊 *每日任务汇总报告* (%s)\n", now.Format("2006-01-02")))
	report.WriteString(fmt.Sprintf("*生成时间*: %s\n\n", timeStr))
	
	// 总体统计
	report.WriteString(fmt.Sprintf("📈 *总体统计*\n"))
	report.WriteString(fmt.Sprintf("• 总任务数: %d\n", stats.TotalTasks))
	report.WriteString(fmt.Sprintf("• 今日完成: %d\n", stats.CompletedToday))
	report.WriteString(fmt.Sprintf("• 今日失败: %d\n", stats.FailedToday))
	report.WriteString(fmt.Sprintf("• 进行中: %d\n", stats.ActiveTasks))
	report.WriteString(fmt.Sprintf("• 已过期: %d\n", stats.OverdueTasks))
	report.WriteString(fmt.Sprintf("• 完成率: %.1f%%\n\n", stats.CompletionRate))
	
	// Token使用
	report.WriteString(fmt.Sprintf("💸 *资源使用*\n"))
	if tokenUsage.Success {
		report.WriteString(fmt.Sprintf("• 今日Token: %s\n", formatNumber(tokenUsage.TotalTokens)))
		report.WriteString(fmt.Sprintf("• 估算成本: ¥%.4f\n", tokenUsage.EstimatedCost))
	} else {
		report.WriteString(fmt.Sprintf("• Token统计: %s\n", tokenUsage.Error))
	}
	report.WriteString("\n")
	
	// 今日完成的任务
	if len(taskDetails.Completed) > 0 {
		report.WriteString(fmt.Sprintf("✅ *今日完成*\n"))
		for _, task := range taskDetails.Completed {
			time := task.CompletedAt.Format("15:04")
			report.WriteString(fmt.Sprintf("• %s (%s)\n", task.Description, time))
		}
		report.WriteString("\n")
	}
	
	// 今日失败的任务
	if len(taskDetails.Failed) > 0 {
		report.WriteString(fmt.Sprintf("❌ *今日失败*\n"))
		for _, task := range taskDetails.Failed {
			time := task.FailedAt.Format("15:04")
			report.WriteString(fmt.Sprintf("• %s (%s)\n", task.Description, time))
			if task.Error != "" {
				report.WriteString(fmt.Sprintf("  错误: %s\n", task.Error))
			}
		}
		report.WriteString("\n")
	}
	
	// 进行中任务
	if len(taskDetails.Active) > 0 {
		report.WriteString(fmt.Sprintf("🔄 *进行中任务*\n"))
		for i, task := range taskDetails.Active {
			emoji := getPriorityEmoji(task.Priority)
			report.WriteString(fmt.Sprintf("%s %s\n", emoji, task.Description))
			if task.Progress != "" {
				report.WriteString(fmt.Sprintf("  进度: %s\n", task.Progress))
			}
			if !task.Deadline.IsZero() {
				deadlineStr := task.Deadline.Format("2006/1/2 15:04")
				report.WriteString(fmt.Sprintf("  截止: %s\n", deadlineStr))
			}
			if i < len(taskDetails.Active)-1 {
				report.WriteString("\n")
			}
		}
		report.WriteString("\n")
	}
	
	// 过期任务
	if len(taskDetails.Overdue) > 0 {
		report.WriteString(fmt.Sprintf("⏰ *过期任务 (需关注)*\n"))
		for _, task := range taskDetails.Overdue {
			deadlineStr := task.Deadline.Format("2006/1/2 15:04")
			report.WriteString(fmt.Sprintf("• %s\n", task.Description))
			report.WriteString(fmt.Sprintf("  应于: %s\n", deadlineStr))
		}
		report.WriteString("\n")
	}
	
	// 建议
	report.WriteString(fmt.Sprintf("🎯 *建议*\n"))
	if stats.OverdueTasks > 0 {
		report.WriteString(fmt.Sprintf("• 优先处理 %d 个过期任务\n", stats.OverdueTasks))
	}
	if stats.CompletionRate < 50 {
		report.WriteString(fmt.Sprintf("• 完成率较低 (%.1f%%)，检查任务可行性\n", stats.CompletionRate))
	}
	if stats.ActiveTasks > 5 {
		report.WriteString(fmt.Sprintf("• 进行中任务较多 (%d)，考虑调整优先级\n", stats.ActiveTasks))
	}
	if stats.TotalTasks == 0 {
		report.WriteString(fmt.Sprintf("• 暂无任务记录，开始创建新任务\n"))
	}
	report.WriteString("\n")
	
	// 明日重点
	report.WriteString(fmt.Sprintf("📅 *明日重点*\n"))
	report.WriteString(fmt.Sprintf("• 继续监控进行中任务\n"))
	if stats.OverdueTasks > 0 {
		report.WriteString(fmt.Sprintf("• 处理 %d 个过期任务\n", stats.OverdueTasks))
	}
	report.WriteString(fmt.Sprintf("• 优化高优先级任务执行\n"))
	report.WriteString(fmt.Sprintf("• 定期检查资源使用情况\n"))
	
	return report.String()
}

// 获取优先级对应的emoji
func getPriorityEmoji(priority string) string {
	switch priority {
	case "critical":
		return "🚨"
	case "high":
		return "🔥"
	case "medium":
		return "📝"
	case "low":
		return "📌"
	default:
		return "📝"
	}
}

// 格式化数字（添加千位分隔符）
func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	var result strings.Builder
	
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	
	return result.String()
}

// 保存报告到文件
func saveReport(report string) string {
	// 确保报告目录存在
	if err := os.MkdirAll(ReportsDir, 0755); err != nil {
		log.Printf("创建报告目录失败: %v", err)
		return ""
	}
	
	now := getBeijingTime()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15-04-05")
	
	filename := fmt.Sprintf("task-report-go-%s-%s.txt", dateStr, timeStr)
	filepath := filepath.Join(ReportsDir, filename)
	
	if err := ioutil.WriteFile(filepath, []byte(report), 0644); err != nil {
		log.Printf("保存报告失败: %v", err)
		return ""
	}
	
	log.Printf("报告已保存: %s", filepath)
	return filepath
}

// 发送报告到Telegram
func sendToTelegram(report string) bool {
	log.Printf("发送报告到Telegram群聊 (ID: %s)...", TelegramChatID)
	
	// 转义报告内容中的特殊字符
	escapedReport := strings.ReplaceAll(report, `"`, `\"`)
	escapedReport = strings.ReplaceAll(escapedReport, "\n", "\\n")
	
	// 构建命令
	cmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", TelegramChatID,
		"--message", escapedReport)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("发送到Telegram失败: %v", err)
		log.Printf("命令输出: %s", output)
		return false
	}
	
	log.Printf("报告已发送到群聊")
	return true
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	log.Println("📊 生成Go版本每日任务汇总报告...")
	
	// 生成报告
	report := generateReport()
	
	// 保存报告
	savedPath := saveReport(report)
	if savedPath == "" {
		log.Fatal("保存报告失败")
	}
	
	// 输出报告到控制台
	fmt.Println(report)
	fmt.Printf("\n✅ 报告生成完成\n")
	fmt.Printf("文件: %s\n", savedPath)
	
	// 发送到Telegram
	if sendToTelegram(report) {
		log.Println("✅ 报告已发送到Telegram群聊")
		os.Exit(0)
	} else {
		log.Println("❌ 发送到Telegram失败")
		os.Exit(1)
	}
}