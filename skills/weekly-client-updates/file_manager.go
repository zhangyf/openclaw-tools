package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileManager 文件管理器
type FileManager struct {
	localDir  string
	cosClient COSClient
	polisher  TextPolisher
}

// NewFileManager 创建新的文件管理器
func NewFileManager(localDir string, cosClient COSClient, polisher TextPolisher) *FileManager {
	return &FileManager{
		localDir:  localDir,
		cosClient: cosClient,
		polisher:  polisher,
	}
}

// GetWeeklyFilePath 获取周报文件路径
func (fm *FileManager) GetWeeklyFilePath(year, week int) string {
	filename := fmt.Sprintf("客户更新汇总_%d年第%02d周.md", year, week)
	return filepath.Join(fm.localDir, filename)
}

// LoadWeeklyReport 加载周报
func (fm *FileManager) LoadWeeklyReport(year, week int, cosPath string) (*WeeklyReport, error) {
	filePath := fm.GetWeeklyFilePath(year, week)
	
	// 先尝试从本地加载
	if data, err := os.ReadFile(filePath); err == nil {
		return fm.parseWeeklyReport(data, year, week, filePath)
	}
	
	// 本地文件不存在，尝试从COS下载
	cosKey := fmt.Sprintf("%s%d/week-%02d.md", cosPath, year, week)
	if data, err := fm.cosClient.DownloadFile(context.Background(), cosKey); err == nil && len(data) > 0 {
		// 保存到本地
		os.MkdirAll(filepath.Dir(filePath), 0755)
		os.WriteFile(filePath, data, 0644)
		return fm.parseWeeklyReport(data, year, week, filePath)
	}
	
	// 都没有，创建新报告
	return fm.createNewWeeklyReport(year, week), nil
}

// createNewWeeklyReport 创建新的周报
func (fm *FileManager) createNewWeeklyReport(year, week int) *WeeklyReport {
	return &WeeklyReport{
		Year:     year,
		Week:     week,
		Clients:  make(map[string]string),
		RawData:  make(map[string][]string),
		FilePath: fm.GetWeeklyFilePath(year, week),
	}
}

// parseWeeklyReport 解析周报文件
func (fm *FileManager) parseWeeklyReport(data []byte, year, week int, filePath string) (*WeeklyReport, error) {
	report := &WeeklyReport{
		Year:     year,
		Week:     week,
		Clients:  make(map[string]string),
		RawData:  make(map[string][]string),
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
				report.Clients[currentClient] = strings.Join(clientContent, "\n")
				clientContent = nil
			}
			
			// 开始新的客户
			currentClient = strings.TrimPrefix(line, "## ")
			currentClient = strings.TrimSpace(currentClient)
			// 去除序号前缀（例如 "1. 客户名" -> "客户名"，也处理 "1. 1. 客户名"）
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
				// 去除数字点号和空格
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
					report.Clients[currentClient] = strings.Join(clientContent, "\n")
					clientContent = nil
					currentClient = ""
					inClientSection = false
				}
			} else if currentClient != "" {
				// 累积客户内容
				clientContent = append(clientContent, line)
			}
		}
	}
	
	// 保存最后一个客户的内容
	if currentClient != "" && len(clientContent) > 0 {
		report.Clients[currentClient] = strings.Join(clientContent, "\n")
	}
	
	return report, nil
}

// ProcessClientUpdate 处理客户更新
func (fm *FileManager) ProcessClientUpdate(report *WeeklyReport, update ClientUpdate) error {
	// 润色内容（目标200字）
	polished, err := fm.polisher.Polish(update.Content, 200)
	if err != nil {
		// 润色失败，使用原始内容
		polished = update.Content
	}
	
	// 记录原始数据
	if report.RawData == nil {
		report.RawData = make(map[string][]string)
	}
	report.RawData[update.Name] = append(report.RawData[update.Name], update.Content)
	
	// 合并到现有内容
	if existing, exists := report.Clients[update.Name]; exists {
		// 客户已存在，追加新内容
		if existing != "" {
			// 检查是否已经包含相似内容（简单去重）
			if !fm.containsSimilarContent(existing, polished) {
				report.Clients[update.Name] = existing + "\n\n" + polished
			} else {
				// 内容相似，更新最后修改时间
				report.Clients[update.Name] = existing + "\n\n> 内容已更新"
			}
		} else {
			report.Clients[update.Name] = polished
		}
	} else {
		// 新客户
		report.Clients[update.Name] = polished
	}
	
	return nil
}

// containsSimilarContent 检查是否包含相似内容（简单实现）
func (fm *FileManager) containsSimilarContent(existing, newContent string) bool {
	// 简单的相似性检查
	existingWords := strings.Fields(existing)
	newWords := strings.Fields(newContent)
	
	if len(existingWords) == 0 || len(newWords) == 0 {
		return false
	}
	
	// 检查是否有显著重叠
	overlap := 0
	for _, word := range newWords {
		if len(word) > 2 { // 只检查较长的词
			if strings.Contains(existing, word) {
				overlap++
			}
		}
	}
	
	// 如果超过30%的词重叠，认为是相似内容
	return float64(overlap)/float64(len(newWords)) > 0.3
}

// SaveWeeklyReport 保存周报
func (fm *FileManager) SaveWeeklyReport(report *WeeklyReport, cosPath string) error {
	// 生成Markdown内容
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
		
		// 添加原始内容引用（可选）
		if rawContents, exists := report.RawData[name]; exists && len(rawContents) > 0 {
			builder.WriteString("> **原始记录**: ")
			for i, raw := range rawContents {
				if i > 0 {
					builder.WriteString("；")
				}
				if len(raw) > 50 {
					builder.WriteString(raw[:50] + "...")
				} else {
					builder.WriteString(raw)
				}
			}
			builder.WriteString("\n\n")
		}
	}
	
	// 添加统计信息
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("**统计**: 本周共更新 %d 个客户", len(report.Clients)))
	
	// 计算总字数
	totalChars := 0
	for _, content := range report.Clients {
		totalChars += len([]rune(content))
	}
	if totalChars > 0 {
		builder.WriteString(fmt.Sprintf("，总字数约 %d 字", totalChars))
	}
	builder.WriteString("\n")
	
	content := builder.String()
	
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(report.FilePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	
	// 保存到本地
	if err := os.WriteFile(report.FilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}
	
	// 上传到COS
	cosKey := fmt.Sprintf("%s%d/week-%02d.md", cosPath, report.Year, report.Week)
	if err := fm.cosClient.UploadFile(context.Background(), cosKey, []byte(content)); err != nil {
		// 记录错误但不中断流程
		fmt.Printf("警告: COS上传失败（文件已保存到本地）: %v\n", err)
	}
	
	return nil
}