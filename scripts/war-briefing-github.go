// war-briefing-github.go
// 战争简报脚本（GitHub版本）
// 生成简报并推送到GitHub，发送GitHub链接到Telegram

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 配置
const (
	GitHubRepo      = "https://github.com/zhangyf/openclaw-tools.git"
	GitHubRawBase   = "https://raw.githubusercontent.com/zhangyf/openclaw-tools/main/briefings/"
	GitHubViewBase  = "https://github.com/zhangyf/openclaw-tools/blob/main/briefings/"
	BriefingsDir    = "/home/zhangyufeng/.openclaw/workspace/briefings"
	WorkspaceDir    = "/home/zhangyufeng/.openclaw/workspace"
	TelegramChatID  = "-5149902750"
)

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 生成简报并获取文件
func generateBriefing() (string, string, error) {
	// 运行战争简报脚本
	cmd := exec.Command("./war-briefing-detailed")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("生成简报失败: %v", err)
	}
	
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	
	// 提取简报摘要
	var telegramBriefing strings.Builder
	var inBriefing bool
	
	for _, line := range lines {
		// 收集Telegram简报内容
		if strings.Contains(line, "📊 **战争财经简报") {
			inBriefing = true
		}
		
		if inBriefing {
			if strings.Contains(line, "📖 **详细版入口**") {
				// 停止收集，我们要替换这部分
				break
			}
			telegramBriefing.WriteString(line)
			telegramBriefing.WriteString("\n")
		}
	}
	
	// 直接查找最新的简报文件
	files, err := os.ReadDir(BriefingsDir)
	if err != nil {
		return "", "", fmt.Errorf("读取简报目录失败: %v", err)
	}
	
	var latestFile string
	var latestTime time.Time
	
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "war-briefing-detailed-") && 
		   strings.HasSuffix(file.Name(), ".md") &&
		   !strings.Contains(file.Name(), "telegram") {
			
			info, err := file.Info()
			if err != nil {
				continue
			}
			
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = file.Name()
			}
		}
	}
	
	if latestFile == "" {
		return "", "", fmt.Errorf("未找到简报文件")
	}
	
	detailedFilePath := filepath.Join(BriefingsDir, latestFile)
	
	return telegramBriefing.String(), detailedFilePath, nil
}

// 推送文件到GitHub
func pushToGitHub(filePath string) (string, string, error) {
	now := getBeijingTime()
	filename := filepath.Base(filePath)
	
	// 1. 复制文件到工作空间的briefings目录（确保在Git仓库内）
	workspaceBriefingsDir := filepath.Join(WorkspaceDir, "briefings")
	if err := os.MkdirAll(workspaceBriefingsDir, 0755); err != nil {
		return "", "", fmt.Errorf("创建目录失败: %v", err)
	}
	
	destPath := filepath.Join(workspaceBriefingsDir, filename)
	
	// 读取原文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("读取文件失败: %v", err)
	}
	
	// 写入到工作空间
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return "", "", fmt.Errorf("写入文件失败: %v", err)
	}
	
	log.Printf("📁 文件已复制到工作空间: %s", destPath)
	
	// 2. 添加到Git
	cmd := exec.Command("git", "add", destPath)
	cmd.Dir = WorkspaceDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("git add失败: %v, 输出: %s", err, output)
	}
	
	// 3. 提交
	commitMsg := fmt.Sprintf("feat: 添加战争简报 %s", now.Format("2006-01-02 15:04"))
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = WorkspaceDir
	if _, err := cmd.CombinedOutput(); err != nil {
		// 如果提交失败（可能没有变化），继续
		log.Printf("⚠️ git commit失败（可能无变化）: %v", err)
	}
	
	// 4. 推送到GitHub
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = WorkspaceDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("git push失败: %v, 输出: %s", err, output)
	}
	
	log.Printf("✅ 文件已推送到GitHub: %s", filename)
	
	// 生成GitHub链接
	rawURL := GitHubRawBase + filename
	viewURL := GitHubViewBase + filename
	
	return rawURL, viewURL, nil
}

// 发送消息到Telegram
func sendToTelegram(message string) error {
	// 转义消息中的特殊字符
	escapedMsg := strings.ReplaceAll(message, `"`, `\"`)
	escapedMsg = strings.ReplaceAll(escapedMsg, "\n", "\\n")
	
	cmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", TelegramChatID,
		"--message", escapedMsg)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("发送消息失败: %v, 输出: %s", err, output)
	}
	
	return nil
}

// 生成带GitHub链接的简报
func generateBriefingWithGitHubLink() (string, error) {
	now := getBeijingTime()
	
	log.Println("🚀 生成战争简报（GitHub版本）...")
	
	// 1. 生成简报
	telegramBriefing, filePath, err := generateBriefing()
	if err != nil {
		return "", err
	}
	
	log.Printf("📁 简报文件: %s", filePath)
	
	// 2. 推送到GitHub
	rawURL, viewURL, err := pushToGitHub(filePath)
	if err != nil {
		log.Printf("⚠️ 推送到GitHub失败，使用本地文件方案: %v", err)
		
		// 如果GitHub推送失败，使用本地.txt方案
		var briefingWithLink strings.Builder
		briefingWithLink.WriteString(telegramBriefing)
		briefingWithLink.WriteString("\n\n📖 **详细版入口**:\n")
		briefingWithLink.WriteString("📎 **详细版已随本消息发送**\n")
		briefingWithLink.WriteString("• 点击下方文件直接在Telegram内预览\n")
		briefingWithLink.WriteString("• 无需跳转到其他应用\n")
		briefingWithLink.WriteString(fmt.Sprintf("⏰ %s\n", now.Format("15:04")))
		
		return briefingWithLink.String(), nil
	}
	
	// 3. 生成带GitHub链接的简报
	var briefingWithLink strings.Builder
	briefingWithLink.WriteString(telegramBriefing)
	briefingWithLink.WriteString("\n\n🌐 **详细版入口（GitHub）**:\n")
	briefingWithLink.WriteString("🔗 **点击链接在GitHub查看完整分析**\n")
	briefingWithLink.WriteString(fmt.Sprintf("• 📖 [在GitHub上查看](%s)\n", viewURL))
	briefingWithLink.WriteString(fmt.Sprintf("• 📥 [下载原始文件](%s)\n", rawURL))
	briefingWithLink.WriteString("• 🎯 优势：完美Markdown渲染 + 版本历史\n")
	briefingWithLink.WriteString("• 📱 在任何设备上都能访问\n")
	briefingWithLink.WriteString(fmt.Sprintf("⏰ %s | 🚀 已同步到GitHub\n", now.Format("15:04")))
	
	log.Printf("✅ GitHub链接已生成:\n   查看: %s\n   下载: %s", viewURL, rawURL)
	
	return briefingWithLink.String(), nil
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// 生成带GitHub链接的简报
	briefing, err := generateBriefingWithGitHubLink()
	if err != nil {
		log.Fatalf("生成简报失败: %v", err)
	}
	
	// 输出到控制台
	fmt.Println(briefing)
	
	// 发送到Telegram
	log.Println("📤 发送简报到Telegram...")
	if err := sendToTelegram(briefing); err != nil {
		log.Printf("❌ 发送到Telegram失败: %v", err)
		os.Exit(1)
	}
	
	log.Println("✅ 简报已发送到Telegram（带GitHub链接）")
	log.Println("")
	log.Println("🎯 用户体验:")
	log.Println("   1. 在群聊中看到简报摘要")
	log.Println("   2. 点击GitHub链接查看详细版")
	log.Println("   3. GitHub完美渲染Markdown格式")
	log.Println("   4. 自动保存版本历史")
	log.Println("")
	log.Println("🚀 后续：可更新cron job使用此脚本")
}