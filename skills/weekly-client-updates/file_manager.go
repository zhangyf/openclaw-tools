package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FileManager 文件管理器
type FileManager struct {
	cosClient COSClient
}

// NewFileManager 创建新的文件管理器（纯COS模式，无本地存储）
func NewFileManager(cosClient COSClient) *FileManager {
	return &FileManager{
		cosClient: cosClient,
	}
}

// GetCOSKey 获取COS上的文件key
func (fm *FileManager) GetCOSKey(cosPath string, year, week int) string {
	return fmt.Sprintf("%s%d/week-%02d.md", cosPath, year, week)
}

// GetLocalizedKey 获取COS上中文字段名的文件key（兼容老格式）
func (fm *FileManager) GetLocalizedKey(cosPath string, year, week int) string {
	return fmt.Sprintf("%s客户更新汇总_%d年第%02d周.md", cosPath, year, week)
}

// LoadWeeklyReport 从COS加载周报，不存在则创建新的
func (fm *FileManager) LoadWeeklyReport(year, week int, cosPath string) (*WeeklyReport, error) {
	cosKey := fm.GetCOSKey(cosPath, year, week)

	// 尝试从COS下载
	data, err := fm.cosClient.DownloadFile(context.Background(), cosKey)
	if err != nil {
		return nil, fmt.Errorf("从COS下载文件失败: %v", err)
	}

	if data != nil && len(data) > 0 {
		// 解析现有周报
		return fm.parseWeeklyReport(data, year, week, cosKey)
	}

	// COS上没有文件，创建新的
	return fm.createNewWeeklyReport(year, week, cosKey), nil
}

// createNewWeeklyReport 创建新的周报
func (fm *FileManager) createNewWeeklyReport(year, week int, filePath string) *WeeklyReport {
	return &WeeklyReport{
		Year:     year,
		Week:     week,
		Clients:  make(map[string]string),
		FilePath: filePath,
	}
}

// parseWeeklyReport 解析周报文件内容
func (fm *FileManager) parseWeeklyReport(data []byte, year, week int, filePath string) (*WeeklyReport, error) {
	report := &WeeklyReport{
		Year:     year,
		Week:     week,
		Clients:  make(map[string]string),
		FilePath: filePath,
	}

	if len(data) == 0 {
		return report, nil
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var currentClient string
	var clientContent []string
	inClientSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "## ") {
			// 保存上一个客户的内容
			if currentClient != "" && len(clientContent) > 0 {
				report.Clients[currentClient] = strings.TrimSpace(strings.Join(clientContent, "\n"))
				clientContent = nil
			}

			// 开始新的客户
			currentClient = strings.TrimPrefix(line, "## ")
			currentClient = strings.TrimSpace(currentClient)
			// 去除序号前缀（例如 "1. 客户名" -> "客户名"）
			for {
				idx := strings.Index(currentClient, ". ")
				if idx <= 0 {
					break
				}
				prefix := currentClient[:idx]
				isNumber := true
				for _, r := range prefix {
					if r < '0' || r > '9' {
						isNumber = false
						break
					}
				}
				if !isNumber {
					break
				}
				currentClient = strings.TrimSpace(currentClient[idx+1:])
				if currentClient == "" {
					break
				}
			}
			inClientSection = true
		} else if inClientSection {
			if line == "" || strings.HasPrefix(line, "# ") {
				// 空行或新的标题，继续下一个客户
				if currentClient != "" && len(clientContent) > 0 {
					report.Clients[currentClient] = strings.TrimSpace(strings.Join(clientContent, "\n"))
					clientContent = nil
					currentClient = ""
					inClientSection = false
				}
			} else if currentClient != "" {
				clientContent = append(clientContent, line)
			}
		}
	}

	// 保存最后一个客户的内容
	if currentClient != "" && len(clientContent) > 0 {
		report.Clients[currentClient] = strings.TrimSpace(strings.Join(clientContent, "\n"))
	}

	return report, nil
}

// ProcessClientUpdate 处理客户更新（原文照录，不做润色）
func (fm *FileManager) ProcessClientUpdate(report *WeeklyReport, update ClientUpdate) {
	// 原文照录，一字不改
	if existing, exists := report.Clients[update.Name]; exists {
		// 客户已存在，追加新内容
		if existing != "" {
			report.Clients[update.Name] = existing + "\n\n" + update.Content
		} else {
			report.Clients[update.Name] = update.Content
		}
	} else {
		// 新客户
		report.Clients[update.Name] = update.Content
	}
}

// SaveWeeklyReport 生成内容并强制上传到COS（不保存本地）
func (fm *FileManager) SaveWeeklyReport(report *WeeklyReport, cosPath string) error {
	content := fm.generateMarkdown(report)

	cosKey := fm.GetCOSKey(cosPath, report.Year, report.Week)

	// 强制上传到COS
	if err := fm.cosClient.UploadFile(context.Background(), cosKey, []byte(content)); err != nil {
		return fmt.Errorf("上传文件到COS失败: %v", err)
	}
	return nil
}

// generateMarkdown 生成Markdown内容
func (fm *FileManager) generateMarkdown(report *WeeklyReport) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# 客户更新汇总（%d年第%02d周）\n\n", report.Year, report.Week))
	builder.WriteString("> 最后更新: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")

	// 按客户名字母顺序排序
	clientNames := make([]string, 0, len(report.Clients))
	for name := range report.Clients {
		clientNames = append(clientNames, name)
	}

	// 简单排序（按拼音或字母）
	for i := 0; i < len(clientNames)-1; i++ {
		for j := i + 1; j < len(clientNames); j++ {
			if clientNames[i] > clientNames[j] {
				clientNames[i], clientNames[j] = clientNames[j], clientNames[i]
			}
		}
	}

	// 输出每个客户（带序号）
	for i, name := range clientNames {
		content := report.Clients[name]
		builder.WriteString(fmt.Sprintf("## %d. %s\n", i+1, name))
		builder.WriteString(content)
		builder.WriteString("\n\n")
	}

	// 添加统计信息
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("**统计**: 本周共更新 %d 个客户", len(report.Clients)))

	totalChars := 0
	for _, content := range report.Clients {
		totalChars += len([]rune(content))
	}
	if totalChars > 0 {
		builder.WriteString(fmt.Sprintf("，总字数约 %d 字", totalChars))
	}
	builder.WriteString("\n")

	return builder.String()
}
