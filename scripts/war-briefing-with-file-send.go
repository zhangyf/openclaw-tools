// war-briefing-with-file-send.go
// 战争简报脚本（直接发送文件版本）
// 生成简报并直接发送详细版文件

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	chatID := "-5149902750" // 张府群聊
	now := getBeijingTime()
	
	log.Println("🚀 生成并发送战争简报（带文件）...")
	
	// 1. 先生成简报文件
	log.Println("📝 生成战争简报...")
	cmd := exec.Command("./war-briefing-detailed")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("生成简报失败: %v", err)
	}
	
	// 2. 找到最新生成的简报文件
	briefingsDir := "/home/zhangyufeng/.openclaw/workspace/briefings"
	
	// 列出所有简报文件，找到最新的
	files, err := os.ReadDir(briefingsDir)
	if err != nil {
		log.Fatalf("读取简报目录失败: %v", err)
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
		log.Fatal("未找到简报文件")
	}
	
	filePath := fmt.Sprintf("%s/%s", briefingsDir, latestFile)
	log.Printf("📁 找到最新简报文件: %s", filePath)
	
	// 3. 生成Telegram简报消息（精简版）
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	
	var telegramBriefing strings.Builder
	var inBriefing bool
	
	for _, line := range lines {
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
	
	// 添加文件入口提示
	telegramBriefing.WriteString("\n\n📎 **详细版已随本消息发送**\n")
	telegramBriefing.WriteString("• 点击下方文件直接查看详细分析\n")
	telegramBriefing.WriteString("• 在Telegram内预览，无需下载\n")
	telegramBriefing.WriteString("• 包含完整数据、策略和风险评估\n")
	telegramBriefing.WriteString(fmt.Sprintf("⏰ %s\n", now.Format("15:04")))
	
	briefingMsg := telegramBriefing.String()
	
	// 4. 先发送简报消息
	log.Println("📤 发送简报消息...")
	sendCmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", chatID,
		"--message", briefingMsg)
	
	if output, err := sendCmd.CombinedOutput(); err != nil {
		log.Printf("发送消息失败: %v, 输出: %s", err, output)
	} else {
		log.Println("✅ 简报消息发送成功")
	}
	
	// 5. 发送详细版文件
	log.Println("📎 发送详细版文件...")
	
	// 构建文件发送命令（不使用caption参数）
	fileCmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", chatID,
		"--media", filePath)
	
	if output, err := fileCmd.CombinedOutput(); err != nil {
		log.Printf("发送文件失败: %v, 输出: %s", err, output)
		
		// 尝试另一种方式：先发送消息，再发送文件
		log.Println("🔄 尝试替代方案：先发送说明消息，再发送文件...")
		
		// 发送说明消息
		fileInfoMsg := fmt.Sprintf("📎 **详细版文件已发送**\n📁 文件名: %s\n⏰ 生成时间: %s\n\n点击下方文件查看完整分析", 
			latestFile, now.Format("15:04"))
		
		infoCmd := exec.Command("openclaw", "message", "send",
			"--channel", "telegram",
			"--target", chatID,
			"--message", fileInfoMsg)
		
		if _, err := infoCmd.CombinedOutput(); err != nil {
			log.Printf("发送文件说明失败: %v", err)
		} else {
			log.Println("✅ 文件说明消息发送成功")
		}
		
		// 再次尝试发送文件
		if _, err := fileCmd.CombinedOutput(); err != nil {
			log.Printf("再次发送文件失败: %v", err)
			os.Exit(1)
		}
	}
	
	log.Println("✅ 详细版文件发送成功")
	log.Println("")
	log.Println("🎯 用户体验:")
	log.Println("   1. 在群聊中看到简报摘要")
	log.Println("   2. 下方就是详细版文件")
	log.Println("   3. 点击文件直接预览")
	log.Println("   4. 无需离开Telegram")
	log.Println("")
	log.Println("🚀 部署: 更新cron job使用此脚本即可")
	os.Exit(0)
}