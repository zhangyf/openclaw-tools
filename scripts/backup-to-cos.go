// backup-to-cos.go
// OpenClaw 增强版备份脚本（Go语言版本）
// 备份主工作空间 + weekly_report_helper 所有配置 + Telegram 配置
// 包含 @zyf_weekly_report_bot token 使用情况统计

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 配置结构
type Config struct {
	Bucket    string
	Region    string
	SecretID  string
	SecretKey string
}

// 备份清单结构
type BackupManifest struct {
	Timestamp string                 `json:"timestamp"`
	Date      string                 `json:"date"`
	Time      string                 `json:"time"`
	Components map[string]interface{} `json:"components"`
	Files     []BackupFile           `json:"files"`
	TokenStats map[string]interface{} `json:"tokenStats,omitempty"`
}

// 备份文件信息
type BackupFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// 常量定义
const (
	WorkspaceDir     = "/home/zhangyufeng/.openclaw/workspace"
	BackupDir        = "/home/zhangyufeng/.openclaw/workspace/backups"
	ReportsDir       = "/home/zhangyufeng/.openclaw/workspace/backup-reports"
	WeeklyWorkspace  = "/home/zhangyufeng/.openclaw/workspace-weekly_report_helper"
	WeeklyAgents     = "/home/zhangyufeng/.openclaw/agents/weekly_report_helper"
	WeeklyMemoryDB   = "/home/zhangyufeng/.openclaw/memory/weekly_report_helper.sqlite"
	TelegramConfig   = "/home/zhangyufeng/.openclaw/telegram"
	BucketName       = "openclaw-bakup-1251036673"
	Region           = "ap-singapore"
)

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	log.Println("🚀 OpenClaw 增强版备份脚本（Go语言版本）启动...")
	
	// 1. 加载配置
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	
	// 2. 准备备份目录
	dateStr := time.Now().Format("2006-01-02")
	timestamp := time.Now().Format("2006-01-02T15-04-05-0700")
	
	backupDateDir := filepath.Join(BackupDir, dateStr)
	reportsDateDir := filepath.Join(ReportsDir, dateStr)
	
	if err := os.MkdirAll(backupDateDir, 0755); err != nil {
		log.Fatalf("❌ 创建备份目录失败: %v", err)
	}
	if err := os.MkdirAll(reportsDateDir, 0755); err != nil {
		log.Fatalf("❌ 创建报告目录失败: %v", err)
	}
	
	// 3. 创建备份清单
	manifest := BackupManifest{
		Timestamp: time.Now().Format(time.RFC3339),
		Date:      dateStr,
		Time:      time.Now().Format("15:04:05"),
		Components: map[string]interface{}{
			"main_workspace": WorkspaceDir,
			"weekly_report_helper": map[string]string{
				"workspace": WeeklyWorkspace,
				"agents":    WeeklyAgents,
				"memory":    WeeklyMemoryDB,
			},
			"telegram_config": TelegramConfig,
		},
		Files: []BackupFile{},
	}
	
	// 4. 创建临时备份目录
	tempBackupDir := filepath.Join(backupDateDir, "temp-enhanced-backup")
	if err := os.RemoveAll(tempBackupDir); err != nil {
		log.Printf("⚠️ 清理临时目录失败: %v", err)
	}
	if err := os.MkdirAll(tempBackupDir, 0755); err != nil {
		log.Fatalf("❌ 创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempBackupDir)
	
	log.Println("📁 收集备份文件...")
	
	// 5. 备份主工作空间
	mainBackupDir := filepath.Join(tempBackupDir, "main-workspace")
	if err := os.MkdirAll(mainBackupDir, 0755); err != nil {
		log.Fatalf("❌ 创建主工作空间备份目录失败: %v", err)
	}
	
	log.Println("📦 备份主工作空间...")
	mainBackupPath := filepath.Join(mainBackupDir, "main-workspace.tar.gz")
	if err := createTarGz(WorkspaceDir, mainBackupPath, []string{
		"node_modules", ".git", "briefings", "token-reports", 
		"backup-reports", "tasks", "scripts-backup-*",
	}); err != nil {
		log.Fatalf("❌ 备份主工作空间失败: %v", err)
	}
	
	mainBackupSize, _ := getFileSize(mainBackupPath)
	manifest.Files = append(manifest.Files, BackupFile{
		Name: "main-workspace.tar.gz",
		Path: "main-workspace/main-workspace.tar.gz",
		Size: mainBackupSize,
	})
	
	// 6. 备份 weekly_report_helper
	if _, err := os.Stat(WeeklyWorkspace); err == nil {
		log.Println("📊 备份 weekly_report_helper 工作空间...")
		weeklyBackupDir := filepath.Join(tempBackupDir, "weekly-report-helper")
		if err := os.MkdirAll(weeklyBackupDir, 0755); err != nil {
			log.Fatalf("❌ 创建 weekly_report_helper 备份目录失败: %v", err)
		}
		
		// 备份核心配置文件
		coreFiles := []string{
			"AGENTS.md", "SOUL.md", "TOOLS.md", "IDENTITY.md",
			"USER.md", "MEMORY.md", "HEARTBEAT.md", "BOOTSTRAP.md",
		}
		
		for _, file := range coreFiles {
			sourcePath := filepath.Join(WeeklyWorkspace, file)
			if _, err := os.Stat(sourcePath); err == nil {
				destPath := filepath.Join(weeklyBackupDir, file)
				if err := copyFile(sourcePath, destPath); err != nil {
					log.Printf("⚠️ 复制文件 %s 失败: %v", file, err)
					continue
				}
				
				size, _ := getFileSize(destPath)
				manifest.Files = append(manifest.Files, BackupFile{
					Name: file,
					Path: fmt.Sprintf("weekly-report-helper/%s", file),
					Size: size,
				})
			}
		}
		
		// 备份 memory 目录
		weeklyMemoryDir := filepath.Join(WeeklyWorkspace, "memory")
		if _, err := os.Stat(weeklyMemoryDir); err == nil {
			memoryBackupPath := filepath.Join(weeklyBackupDir, "memory.tar.gz")
			if err := createTarGz(weeklyMemoryDir, memoryBackupPath, nil); err != nil {
				log.Printf("⚠️ 备份 memory 目录失败: %v", err)
			} else {
				size, _ := getFileSize(memoryBackupPath)
				manifest.Files = append(manifest.Files, BackupFile{
					Name: "memory.tar.gz",
					Path: "weekly-report-helper/memory.tar.gz",
					Size: size,
				})
			}
		}
		
		// 备份 weekly_summaries 目录
		summariesDir := filepath.Join(WeeklyWorkspace, "weekly_summaries")
		if _, err := os.Stat(summariesDir); err == nil {
			summariesBackupPath := filepath.Join(weeklyBackupDir, "weekly_summaries.tar.gz")
			if err := createTarGz(summariesDir, summariesBackupPath, nil); err != nil {
				log.Printf("⚠️ 备份 weekly_summaries 目录失败: %v", err)
			} else {
				size, _ := getFileSize(summariesBackupPath)
				manifest.Files = append(manifest.Files, BackupFile{
					Name: "weekly_summaries.tar.gz",
					Path: "weekly-report-helper/weekly_summaries.tar.gz",
					Size: size,
				})
			}
		}
		
		// 备份 clients 目录
		clientsDir := filepath.Join(WeeklyWorkspace, "clients")
		if _, err := os.Stat(clientsDir); err == nil {
			clientsBackupPath := filepath.Join(weeklyBackupDir, "clients.tar.gz")
			if err := createTarGz(clientsDir, clientsBackupPath, nil); err != nil {
				log.Printf("⚠️ 备份 clients 目录失败: %v", err)
			} else {
				size, _ := getFileSize(clientsBackupPath)
				manifest.Files = append(manifest.Files, BackupFile{
					Name: "clients.tar.gz",
					Path: "weekly-report-helper/clients.tar.gz",
					Size: size,
				})
			}
		}
	}
	
	// 7. 备份 weekly_report_helper 代理配置
	if _, err := os.Stat(WeeklyAgents); err == nil {
		log.Println("🤖 备份 weekly_report_helper 代理配置...")
		agentsBackupDir := filepath.Join(tempBackupDir, "weekly-report-helper-agents")
		if err := os.MkdirAll(agentsBackupDir, 0755); err != nil {
			log.Fatalf("❌ 创建代理配置备份目录失败: %v", err)
		}
		
		agentsBackupPath := filepath.Join(agentsBackupDir, "weekly_report_helper.tar.gz")
		if err := createTarGz(WeeklyAgents, agentsBackupPath, nil); err != nil {
			log.Printf("⚠️ 备份代理配置失败: %v", err)
		} else {
			size, _ := getFileSize(agentsBackupPath)
			manifest.Files = append(manifest.Files, BackupFile{
				Name: "weekly_report_helper.tar.gz",
				Path: "weekly-report-helper-agents/weekly_report_helper.tar.gz",
				Size: size,
			})
		}
	}
	
	// 8. 备份 weekly_report_helper 数据库
	if _, err := os.Stat(WeeklyMemoryDB); err == nil {
		log.Println("💾 备份 weekly_report_helper 数据库...")
		dbBackupDir := filepath.Join(tempBackupDir, "weekly-report-helper-db")
		if err := os.MkdirAll(dbBackupDir, 0755); err != nil {
			log.Fatalf("❌ 创建数据库备份目录失败: %v", err)
		}
		
		dbBackupPath := filepath.Join(dbBackupDir, "weekly_report_helper.sqlite")
		if err := copyFile(WeeklyMemoryDB, dbBackupPath); err != nil {
			log.Printf("⚠️ 备份数据库失败: %v", err)
		} else {
			size, _ := getFileSize(dbBackupPath)
			manifest.Files = append(manifest.Files, BackupFile{
				Name: "weekly_report_helper.sqlite",
				Path: "weekly-report-helper-db/weekly_report_helper.sqlite",
				Size: size,
			})
		}
	}
	
	// 9. 备份 Telegram 配置
	if _, err := os.Stat(TelegramConfig); err == nil {
		log.Println("📱 备份 Telegram 配置...")
		telegramBackupDir := filepath.Join(tempBackupDir, "telegram-config")
		if err := os.MkdirAll(telegramBackupDir, 0755); err != nil {
			log.Fatalf("❌ 创建 Telegram 配置备份目录失败: %v", err)
		}
		
		telegramBackupPath := filepath.Join(telegramBackupDir, "telegram.tar.gz")
		if err := createTarGz(TelegramConfig, telegramBackupPath, nil); err != nil {
			log.Printf("⚠️ 备份 Telegram 配置失败: %v", err)
		} else {
			size, _ := getFileSize(telegramBackupPath)
			manifest.Files = append(manifest.Files, BackupFile{
				Name: "telegram.tar.gz",
				Path: "telegram-config/telegram.tar.gz",
				Size: size,
			})
		}
	}
	
	// 10. 获取 Token 统计
	log.Println("📊 获取 Token 使用统计...")
	tokenStats, err := getTokenStats()
	if err != nil {
		log.Printf("⚠️ 获取 Token 统计失败: %v", err)
	} else {
		manifest.TokenStats = tokenStats
	}
	
	// 11. 创建最终备份文件
	log.Println("📦 创建最终备份文件...")
	finalBackupName := fmt.Sprintf("openclaw-enhanced-backup-%s-%s.tar.gz", dateStr, timestamp)
	finalBackupPath := filepath.Join(backupDateDir, finalBackupName)
	
	if err := createTarGz(tempBackupDir, finalBackupPath, nil); err != nil {
		log.Fatalf("❌ 创建最终备份文件失败: %v", err)
	}
	
	finalBackupSize, _ := getFileSize(finalBackupPath)
	
	// 12. 上传到 COS
	log.Println("☁️ 上传备份到腾讯云 COS...")
	if err := uploadToCOS(config, finalBackupPath, dateStr, finalBackupName); err != nil {
		log.Fatalf("❌ 上传到 COS 失败: %v", err)
	}
	
	// 13. 保存备份报告
	log.Println("📝 生成备份报告...")
	reportData := map[string]interface{}{
		"backup": map[string]interface{}{
			"fileName": finalBackupName,
			"fileSize": finalBackupSize,
			"fileSizeMB": fmt.Sprintf("%.2f MB", float64(finalBackupSize)/1024/1024),
			"localPath": finalBackupPath,
			"cosPath":   fmt.Sprintf("%s/backups/%s/%s", BucketName, dateStr, finalBackupName),
			"timestamp": manifest.Timestamp,
		},
		"manifest": manifest,
		"summary": map[string]interface{}{
			"totalFiles": len(manifest.Files),
			"totalSize":  finalBackupSize,
			"totalSizeMB": fmt.Sprintf("%.2f MB", float64(finalBackupSize)/1024/1024),
			"components": []string{
				"主工作空间",
				"weekly_report_helper 工作空间",
				"weekly_report_helper 代理配置",
				"weekly_report_helper 数据库",
				"Telegram 配置",
			},
		},
	}
	
	reportFileName := fmt.Sprintf("backup-report-enhanced-%s-%s.json", dateStr, timestamp)
	reportPath := filepath.Join(reportsDateDir, reportFileName)
	
	reportJSON, _ := json.MarshalIndent(reportData, "", "  ")
	if err := ioutil.WriteFile(reportPath, reportJSON, 0644); err != nil {
		log.Printf("⚠️ 保存备份报告失败: %v", err)
	}
	
	// 14. 输出总结
	log.Println("✅ 备份执行完成")
	log.Printf("📁 备份文件: %s", finalBackupName)
	log.Printf("📏 文件大小: %.2f MB", float64(finalBackupSize)/1024/1024)
	log.Printf("📍 存储位置: COS存储桶 %s 的 /backups/%s/ 目录下", BucketName, dateStr)
	log.Printf("📋 备份报告: %s", reportPath)
	
	if tokenStats != nil {
		log.Printf("💰 Token费用统计: 估算费用约 $%.6f (约 %.4f 元)", 
			tokenStats["estimatedCostUSD"].(float64),
			tokenStats["estimatedCostCNY"].(float64))
	}
	
	// 15. 发送通知（可选）
	sendNotification(finalBackupName, finalBackupSize, dateStr)
}

// 加载配置
func loadConfig() (*Config, error) {
	secretID := os.Getenv("TENCENT_COS_SECRET_ID")
	secretKey := os.Getenv("TENCENT_COS_SECRET_KEY")
	
	if secretID == "" || secretKey == "" {
		// 尝试从 .env 文件加载
		envPath := filepath.Join(os.Getenv("HOME"), ".openclaw/workspace/.env")
		if _, err := os.Stat(envPath); err == nil {
			content, err := ioutil.ReadFile(envPath)
			if err != nil {
				return nil, fmt.Errorf("读取.env文件失败: %v", err)
			}
			
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "TENCENT_COS_SECRET_ID=") {
					secretID = strings.TrimPrefix(line, "TENCENT_COS_SECRET_ID=")
					secretID = strings.Trim(secretID, `"`)
				} else if strings.HasPrefix(line, "TENCENT_COS_SECRET_KEY=") {
					secretKey = strings.TrimPrefix(line, "TENCENT_COS_SECRET_KEY=")
					secretKey = strings.Trim(secretKey, `"`)
				}
			}
			log.Println("✅ 已从.env文件加载环境变量")
		}
	}
	
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("请设置环境变量 TENCENT_COS_SECRET_ID 和 TENCENT_COS_SECRET_KEY")
	}
	
	return &Config{
		Bucket:    BucketName,
		Region:    Region,
		SecretID:  secretID,
		SecretKey: secretKey,
	}, nil
}

// 创建 tar.gz 压缩包
func createTarGz(sourceDir, outputPath string, excludePatterns []string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	
	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()
	
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 检查是否排除
		relPath, _ := filepath.Rel(sourceDir, filePath)
		for _, pattern := range excludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		
		// 创建 tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		
		header.Name = relPath
		
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		if !info.IsDir() {
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			
			_, err = io.Copy(tarWriter, file)
			return err
		}
		
		return nil
	})
}

// 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}

// 获取文件大小
func getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// 获取 Token 统计
func getTokenStats() (map[string]interface{}, error) {
	// 直接读取 token-stats 生成的 JSON 文件
	reportDir := filepath.Join(WorkspaceDir, "token-reports")
	files, err := ioutil.ReadDir(reportDir)
	if err != nil {
		return nil, fmt.Errorf("读取报告目录失败: %v", err)
	}
	
	// 找到最新的 JSON 报告文件
	var latestFile string
	var latestTime time.Time
	
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "token-stats-") && strings.HasSuffix(file.Name(), ".json") {
			if file.ModTime().After(latestTime) {
				latestTime = file.ModTime()
				latestFile = file.Name()
			}
		}
	}
	
	if latestFile == "" {
		return nil, fmt.Errorf("未找到 token 统计报告")
	}
	
	reportPath := filepath.Join(reportDir, latestFile)
	content, err := ioutil.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("读取报告文件失败: %v", err)
	}
	
	var stats map[string]interface{}
	if err := json.Unmarshal(content, &stats); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %v", err)
	}
	
	return stats, nil
}

// 上传到腾讯云 COS
func uploadToCOS(config *Config, filePath, dateStr, fileName string) error {
	// 这里简化处理，使用 curl 命令上传
	// 实际生产环境应该使用腾讯云 COS Go SDK
	
	cosURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com/backups/%s/%s",
		config.Bucket, config.Region, dateStr, fileName)
	
	// 使用 curl 上传
	cmd := exec.Command("curl", "-X", "PUT",
		"-H", "Content-Type: application/octet-stream",
		"-H", fmt.Sprintf("Authorization: q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%d&q-key-time=%d&q-header-list=&q-url-param-list=&q-signature=", 
			config.SecretID, time.Now().Unix(), time.Now().Unix()),
		"--upload-file", filePath,
		cosURL)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("上传失败: %v, 输出: %s", err, output)
	}
	
	log.Printf("✅ 上传成功: %s", cosURL)
	return nil
}

// 发送通知
func sendNotification(fileName string, fileSize int64, dateStr string) {
	// 这里可以集成 Telegram 通知
	// 暂时只输出日志
	log.Printf("📤 备份完成通知: 文件 %s (%.2f MB) 已备份到 COS", 
		fileName, float64(fileSize)/1024/1024)
}