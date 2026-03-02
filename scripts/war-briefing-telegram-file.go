// war-briefing-telegram-file.go
// 战争简报脚本（Telegram文件发送版本）
// 生成简报并发送详细版文件到Telegram

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Telegram发送消息的响应
type TelegramResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// 获取当前北京时间
func getBeijingTime() time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc)
}

// 发送文件到Telegram
func sendFileToTelegram(filePath, chatID, caption string) (bool, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, fmt.Errorf("文件不存在: %s", filePath)
	}
	
	// 构建发送命令
	cmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", chatID,
		"--media", filePath,
		"--caption", caption)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("发送文件失败: %v, 输出: %s", err, output)
	}
	
	// 解析响应
	var resp TelegramResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return false, fmt.Errorf("解析响应失败: %v", err)
	}
	
	return resp.OK, nil
}

// 发送消息到Telegram
func sendMessageToTelegram(message, chatID string) (bool, error) {
	// 转义消息中的特殊字符
	escapedMsg := strings.ReplaceAll(message, `"`, `\"`)
	escapedMsg = strings.ReplaceAll(escapedMsg, "\n", "\\n")
	
	cmd := exec.Command("openclaw", "message", "send",
		"--channel", "telegram",
		"--target", chatID,
		"--message", escapedMsg)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("发送消息失败: %v, 输出: %s", err, output)
	}
	
	var resp TelegramResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return false, fmt.Errorf("解析响应失败: %v", err)
	}
	
	return resp.OK, nil
}

// 生成带文件链接的简报
func generateBriefingWithFileLink() (string, string, error) {
	// 运行原有的战争简报脚本
	cmd := exec.Command("./war-briefing-detailed")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("执行简报脚本失败: %v", err)
	}
	
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	
	// 提取文件路径和Telegram简报
	var telegramBriefing strings.Builder
	var detailedFilePath string
	var inTelegramBriefing bool
	
	for _, line := range lines {
		// 查找文件路径 - 从查看命令中提取
		if strings.Contains(line, "查看: `cat ") && detailedFilePath == "" {
			start := strings.Index(line, "cat ") + 4
			end := strings.LastIndex(line, "`")
			if start > 0 && end > start {
				detailedFilePath = line[start:end]
			}
		}
		
		// 收集Telegram简报内容（直到详细版入口之前）
		if strings.Contains(line, "📊 **战争财经简报") {
			inTelegramBriefing = true
		}
		
		if inTelegramBriefing {
			if strings.Contains(line, "📖 **详细版入口**") {
				// 停止收集，我们要替换这部分
				break
			}
			telegramBriefing.WriteString(line)
			telegramBriefing.WriteString("\n")
		}
	}
	
	if detailedFilePath == "" {
		return "", "", fmt.Errorf("未找到详细版文件路径")
	}
	
	// 在简报末尾添加文件入口
	telegramBriefing.WriteString("\n\n📖 **详细版入口**:\n")
	telegramBriefing.WriteString("📎 **点击下方文件直接查看详细分析**\n")
	telegramBriefing.WriteString("• 文件已随本消息一起发送\n")
	telegramBriefing.WriteString("• 在Telegram中点击即可预览\n")
	telegramBriefing.WriteString("• 无需下载，直接查看\n")
	
	timeStr := getBeijingTime().Format("15:04")
	telegramBriefing.WriteString(fmt.Sprintf("⏰ 生成时间: %s\n", timeStr))
	
	return telegramBriefing.String(), detailedFilePath, nil
}

// 主函数
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	// 配置
	chatID := "-5149902750" // 张府群聊
	
	log.Println("📊 生成带文件链接的战争简报...")
	
	// 生成简报和文件路径
	telegramBriefing, filePath, err := generateBriefingWithFileLink()
	if err != nil {
		log.Fatalf("生成简报失败: %v", err)
	}
	
	log.Printf("📁 详细版文件: %s", filePath)
	
	// 1. 先发送简报消息
	log.Println("📤 发送简报消息到Telegram...")
	success, err := sendMessageToTelegram(telegramBriefing, chatID)
	if err != nil {
		log.Printf("❌ 发送简报消息失败: %v", err)
	} else if success {
		log.Println("✅ 简报消息发送成功")
	}
	
	// 2. 发送详细版文件
	log.Println("📎 发送详细版文件到Telegram...")
	
	// 读取文件内容作为caption
	fileContent, _ := ioutil.ReadFile(filePath)
	caption := fmt.Sprintf("📊 战争财经详细版简报\n⏰ %s\n📖 点击上方预览按钮查看完整内容", 
		getBeijingTime().Format("2006-01-02 15:04"))
	
	// 只取前200字符作为caption（Telegram限制）
	if len(fileContent) > 200 {
		caption += "\n\n" + string(fileContent[:200]) + "..."
	}
	
	success, err = sendFileToTelegram(filePath, chatID, caption)
	if err != nil {
		log.Printf("❌ 发送文件失败: %v", err)
		os.Exit(1)
	}
	
	if success {
		log.Println("✅ 详细版文件发送成功")
		log.Println("🎯 用户现在可以：")
		log.Println("   1. 在群聊中看到简报摘要")
		log.Println("   2. 点击下方文件直接查看详细版")
		log.Println("   3. 在Telegram内预览，无需下载")
		os.Exit(0)
	} else {
		log.Println("❌ 文件发送失败")
		os.Exit(1)
	}
}