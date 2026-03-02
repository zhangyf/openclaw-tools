// war-briefing-final.go
// 战争简报脚本（最终版本）
// 优先使用GitHub Gist，失败时使用.txt文件备用

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 配置
const (
	TelegramChatID = "-5149902750"
	BriefingsDir   = "/home/zhangyufeng/.openclaw/workspace/briefings"
)

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 生成简报
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
		if strings.Contains(line, "📊 **战争财经简报") {
			inBriefing = true
		}
		
		if inBriefing {
			if strings.Contains(line, "📖 **详细版入口**") {
				break
			}
			telegramBriefing.WriteString(line)
			telegramBriefing.WriteString("\n")
		}
	}
	
	// 查找最新的简报文件
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
	
	filePath := filepath.Join(BriefingsDir, latestFile)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("读取文件失败: %v", err)
	}
	
	return telegramBriefing.String(), string(content), nil
}

// 创建Gist
func createGist(content string) (string, error) {
	now := getBeijingTime()
	
	// 检查gh CLI
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("未安装GitHub CLI")
	}
	
	// 创建临时文件
	tmpFile, err := ioutil.TempFile("", "briefing-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(content); err != nil {
		return "", err
	}
	tmpFile.Close()
	
	// 创建Gist
	description := fmt.Sprintf("战争财经简报 - %s", now.Format("2006-01-02 15:04"))
	cmd := exec.Command("gh", "gist", "create",
		"--public",
		"--desc", description,
		tmpFile.Name())
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("创建Gist失败: %v", err)
	}
	
	gistURL := strings.TrimSpace(string(output))
	log.Printf("✅ Gist创建成功: %s", gistURL)
	return gistURL, nil
}

// 发送.txt文件到Telegram
func sendTxtFile(content string) error {
	// 创建临时.txt文件
	tmpFile, err := ioutil.TempFile("", "briefing-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(content); err != nil {
		return err
	}
	tmpFile.Close()
	
	// 发送文件
	cmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", TelegramChatID,
		"--media", tmpFile.Name())
	
	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}
	
	log.Println("✅ .txt文件发送成功")
	return nil
}

// 发送消息到Telegram
func sendMessage(message string) error {
	escapedMsg := strings.ReplaceAll(message, `"`, `\"`)
	escapedMsg = strings.ReplaceAll(escapedMsg, "\n", "\\n")
	
	cmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", TelegramChatID,
		"--message", escapedMsg)
	
	_, err := cmd.CombinedOutput()
	return err
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	now := getBeijingTime()
	
	log.Println("🚀 生成战争简报（最终版本）...")
	
	// 1. 生成简报
	telegramBriefing, content, err := generateBriefing()
	if err != nil {
		log.Fatalf("生成简报失败: %v", err)
	}
	
	// 2. 尝试创建Gist
	var finalBriefing strings.Builder
	finalBriefing.WriteString(telegramBriefing)
	finalBriefing.WriteString("\n\n")
	
	gistURL, err := createGist(content)
	if err == nil {
		// Gist方案成功
		finalBriefing.WriteString("🌐 **详细版入口（GitHub Gist）**:\n")
		finalBriefing.WriteString("🔗 **点击链接在GitHub Gist查看完整分析**\n")
		finalBriefing.WriteString(fmt.Sprintf("• 📖 [在Gist上查看](%s)\n", gistURL))
		finalBriefing.WriteString("• 🎯 优势：短链接 + 完美Markdown渲染\n")
		finalBriefing.WriteString("• 🔒 公开Gist，无需登录即可查看\n")
		finalBriefing.WriteString("• 📱 在任何设备上都能访问\n")
		finalBriefing.WriteString(fmt.Sprintf("⏰ %s | 🚀 已发布到GitHub Gist\n", now.Format("15:04")))
		
		log.Println("✅ 使用Gist方案")
	} else {
		// Gist失败，使用.txt文件方案
		log.Printf("⚠️ Gist创建失败，使用.txt文件方案: %v", err)
		
		finalBriefing.WriteString("📎 **详细版入口**:\n")
		finalBriefing.WriteString("📁 **详细版已随本消息发送**\n")
		finalBriefing.WriteString("• 点击下方.txt文件直接在Telegram内预览\n")
		finalBriefing.WriteString("• 无需跳转到其他应用\n")
		finalBriefing.WriteString("• 内容完整，格式为纯文本\n")
		finalBriefing.WriteString(fmt.Sprintf("⏰ %s\n", now.Format("15:04")))
		
		// 发送.txt文件
		if err := sendTxtFile(content); err != nil {
			log.Printf("❌ 发送.txt文件失败: %v", err)
		}
		
		log.Println("✅ 使用.txt文件方案（备用）")
	}
	
	// 3. 发送简报消息
	briefingMsg := finalBriefing.String()
	fmt.Println(briefingMsg)
	
	log.Println("📤 发送简报到Telegram...")
	if err := sendMessage(briefingMsg); err != nil {
		log.Printf("❌ 发送消息失败: %v", err)
		os.Exit(1)
	}
	
	log.Println("✅ 简报发送完成")
	log.Println("")
	log.Println("🎯 部署策略:")
	log.Println("   1. 优先使用GitHub Gist（最佳体验）")
	log.Println("   2. 失败时自动使用.txt文件（可靠备用）")
	log.Println("   3. 确保用户总能查看详细版")
	log.Println("")
	log.Println("🚀 建议：更新cron job使用此最终版本")
}