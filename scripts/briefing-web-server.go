// briefing-web-server.go
// 简报Web服务器，提供详细版简报的Web访问

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 配置
const (
	Port        = 8080
	BaseURL     = "http://localhost:8080"
	BriefingsDir = "/home/zhangyufeng/.openclaw/workspace/briefings"
)

// 将Markdown转换为HTML
func markdownToHTML(markdown string) string {
	// 简单的Markdown到HTML转换
	html := strings.ReplaceAll(markdown, "\n", "<br>")
	html = strings.ReplaceAll(html, "# ", "<h1>")
	html = strings.ReplaceAll(html, "\n# ", "</h1><h1>")
	html = strings.ReplaceAll(html, "## ", "<h2>")
	html = strings.ReplaceAll(html, "\n## ", "</h2><h2>")
	html = strings.ReplaceAll(html, "### ", "<h3>")
	html = strings.ReplaceAll(html, "\n### ", "</h3><h3>")
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "*", "<em>")
	html = strings.ReplaceAll(html, "`", "<code>")
	html = strings.ReplaceAll(html, "---", "<hr>")
	
	// 处理列表
	html = strings.ReplaceAll(html, "\n• ", "<li>")
	html = strings.ReplaceAll(html, "\n  • ", "<li>")
	
	return html
}

// 生成HTML页面
func generateHTMLPage(title, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            border-radius: 10px;
            padding: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1, h2, h3 {
            color: #333;
            margin-top: 1.5em;
        }
        h1 { font-size: 24px; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { font-size: 20px; color: #555; }
        h3 { font-size: 18px; color: #666; }
        p { color: #444; margin: 1em 0; }
        strong { color: #d32f2f; }
        em { color: #1976d2; }
        code {
            background: #f1f1f1;
            padding: 2px 6px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
        }
        hr {
            border: none;
            border-top: 1px solid #ddd;
            margin: 2em 0;
        }
        .timestamp {
            color: #888;
            font-size: 14px;
            text-align: right;
            margin-top: 20px;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
            padding: 10px 20px;
            background: #4CAF50;
            color: white;
            text-decoration: none;
            border-radius: 5px;
        }
        .back-link:hover {
            background: #45a049;
        }
        @media (max-width: 600px) {
            body { padding: 10px; }
            .container { padding: 15px; }
        }
    </style>
</head>
<body>
    <div class="container">
        %s
        <div class="timestamp">
            生成时间: %s
        </div>
        <a href="/" class="back-link">返回简报列表</a>
    </div>
</body>
</html>`, title, content, time.Now().Format("2006-01-02 15:04:05"))
}

// 处理简报文件请求
func briefingHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		// 列出所有简报文件
		files, err := ioutil.ReadDir(BriefingsDir)
		if err != nil {
			http.Error(w, "无法读取简报目录", http.StatusInternalServerError)
			return
		}
		
		var fileList strings.Builder
		fileList.WriteString("<h1>📊 战争财经简报存档</h1>")
		fileList.WriteString("<p>点击以下链接查看详细版简报：</p>")
		fileList.WriteString("<ul>")
		
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".md") && strings.Contains(file.Name(), "war-briefing-detailed") {
				fileList.WriteString(fmt.Sprintf(
					`<li><a href="/briefing?file=%s">%s</a></li>`,
					file.Name(), file.Name(),
				))
			}
		}
		
		fileList.WriteString("</ul>")
		fileList.WriteString("<p>最新简报会显示在Telegram群聊中，点击链接即可查看详细内容。</p>")
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, generateHTMLPage("简报存档", fileList.String()))
		return
	}
	
	// 读取简报文件
	filepath := filepath.Join(BriefingsDir, filename)
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		http.Error(w, "简报文件不存在", http.StatusNotFound)
		return
	}
	
	// 转换为HTML
	htmlContent := markdownToHTML(string(content))
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, generateHTMLPage(filename, htmlContent))
}

// 启动Web服务器
func startWebServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			briefingHandler(w, r)
			return
		}
		http.NotFound(w, r)
	})
	
	http.HandleFunc("/briefing", briefingHandler)
	
	addr := fmt.Sprintf(":%d", Port)
	log.Printf("📡 简报Web服务器启动: %s", BaseURL)
	log.Printf("📁 简报目录: %s", BriefingsDir)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// 获取最新简报文件
func getLatestBriefing() (string, error) {
	files, err := ioutil.ReadDir(BriefingsDir)
	if err != nil {
		return "", err
	}
	
	var latestFile string
	var latestTime time.Time
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".md") && strings.Contains(file.Name(), "war-briefing-detailed") {
			fileTime := file.ModTime()
			if fileTime.After(latestTime) {
				latestTime = fileTime
				latestFile = file.Name()
			}
		}
	}
	
	if latestFile == "" {
		return "", fmt.Errorf("未找到简报文件")
	}
	
	return latestFile, nil
}

// 生成Telegram简报（带Web链接）
func generateTelegramBriefingWithLink() (string, string, error) {
	latestFile, err := getLatestBriefing()
	if err != nil {
		return "", "", err
	}
	
	// 读取简报内容
	filepath := filepath.Join(BriefingsDir, latestFile)
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", "", err
	}
	
	// 提取前几行作为摘要
	lines := strings.Split(string(content), "\n")
	var summary strings.Builder
	
	for i := 0; i < min(10, len(lines)); i++ {
		summary.WriteString(lines[i])
		summary.WriteString("\n")
	}
	
	// 生成Web链接
	webLink := fmt.Sprintf("%s/briefing?file=%s", BaseURL, latestFile)
	
	// 生成Telegram消息
	var telegramMsg strings.Builder
	telegramMsg.WriteString("📊 **战争财经简报**\n\n")
	telegramMsg.WriteString(summary.String())
	telegramMsg.WriteString("\n...\n\n")
	telegramMsg.WriteString("📖 **查看完整详细版**:\n")
	telegramMsg.WriteString(fmt.Sprintf("🔗 [点击这里查看详细分析](%s)\n", webLink))
	telegramMsg.WriteString(fmt.Sprintf("📁 文件: `%s`\n", latestFile))
	telegramMsg.WriteString(fmt.Sprintf("⏰ 生成时间: %s\n", time.Now().Format("15:04")))
	
	return telegramMsg.String(), webLink, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 主函数
func main() {
	// 检查命令行参数
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			startWebServer()
			return
		case "generate":
			msg, link, err := generateTelegramBriefingWithLink()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(msg)
			fmt.Printf("\n🔗 Web链接: %s\n", link)
			return
		case "latest":
			file, err := getLatestBriefing()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(file)
			return
		}
	}
	
	// 默认显示帮助
	fmt.Println("简报Web服务器使用说明:")
	fmt.Println("  serve     - 启动Web服务器")
	fmt.Println("  generate  - 生成带Web链接的Telegram简报")
	fmt.Println("  latest    - 获取最新简报文件名")
	fmt.Println("")
	fmt.Printf("Web服务器地址: %s\n", BaseURL)
	fmt.Printf("简报目录: %s\n", BriefingsDir)
}