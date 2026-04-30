package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 程序版本
const Version = "2.1.0"

func main() {
	// 解析命令行参数
	var (
		clientName    string
		content       string
		clientsJSON   string
		bucket        string
		region        string
		secretID      string
		secretKey     string
		cosPath       string
		year          int
		week          int
		showHelp      bool
		showVersion   bool
		listMode      bool
		historyClient string
	)

	flag.StringVar(&clientName, "client", "", "客户名称")
	flag.StringVar(&content, "content", "", "更新内容")
	flag.StringVar(&clientsJSON, "clients", "", "多个客户更新（JSON格式）")
	flag.StringVar(&bucket, "bucket", "", "COS桶名称（必需）")
	flag.StringVar(&region, "region", "", "COS区域，默认ap-beijing")
	flag.StringVar(&secretID, "secret-id", "", "腾讯云SecretId")
	flag.StringVar(&secretKey, "secret-key", "", "腾讯云SecretKey")
	flag.StringVar(&cosPath, "cos-path", "", "COS上传路径，默认weekly-updates/")
	flag.IntVar(&year, "year", 0, "年份（默认当前年）")
	flag.IntVar(&week, "week", 0, "周数（默认当前周）")
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.BoolVar(&listMode, "list", false, "列出本周或指定周的客户情况汇总")
	flag.StringVar(&historyClient, "history", "", "查询指定客户的历史更新情况")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "每周客户更新管理工具 v%s\n\n", Version)
		fmt.Fprintf(os.Stderr, "使用方法：\n")
		fmt.Fprintf(os.Stderr, "  %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项：\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例：\n")
		fmt.Fprintf(os.Stderr, "  单个客户更新：\n")
		fmt.Fprintf(os.Stderr, "    %s --client \"客户A\" --content \"项目进展...\" --bucket my-bucket\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  多个客户更新：\n")
		fmt.Fprintf(os.Stderr, "    %s --clients '[{\"name\":\"客户A\",\"content\":\"内容A\"}]' --bucket my-bucket\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  列出本周客户汇总：\n")
		fmt.Fprintf(os.Stderr, "    %s --list --bucket my-bucket\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  列出指定周客户汇总：\n")
		fmt.Fprintf(os.Stderr, "    %s --list --year 2026 --week 14 --bucket my-bucket\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  查询客户历史更新：\n")
		fmt.Fprintf(os.Stderr, "    %s --history \"客户A\" --bucket my-bucket\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n环境变量：\n")
		fmt.Fprintf(os.Stderr, "  WEEKLY_CLIENT_UPDATE_SECRET_ID    腾讯云SecretId\n")
		fmt.Fprintf(os.Stderr, "  WEEKLY_CLIENT_UPDATE_SECRET_KEY   腾讯云SecretKey\n")
		fmt.Fprintf(os.Stderr, "  WEEKLY_CLIENT_UPDATE_BUCKET       默认COS桶名称\n")
		fmt.Fprintf(os.Stderr, "  WEEKLY_CLIENT_UPDATE_REGION       默认COS区域\n")
		fmt.Fprintf(os.Stderr, "  WEEKLY_CLIENT_UPDATE_COS_PATH     COS上传路径\n")
	}

	flag.Parse()

	// 显示帮助或版本信息
	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		fmt.Printf("每周客户更新管理工具 v%s\n", Version)
		return
	}

	// 参数验证
	if bucket == "" {
		bucket = os.Getenv("WEEKLY_CLIENT_UPDATE_BUCKET")
		if bucket == "" {
			log.Fatal("错误: 必须指定COS桶名称（--bucket 或设置 WEEKLY_CLIENT_UPDATE_BUCKET 环境变量）")
		}
	}

	if region == "" {
		region = os.Getenv("WEEKLY_CLIENT_UPDATE_REGION")
		if region == "" {
			region = "ap-beijing"
		}
	}

	// 设置COS上传路径
	if cosPath == "" {
		cosPath = os.Getenv("WEEKLY_CLIENT_UPDATE_COS_PATH")
		if cosPath == "" {
			cosPath = "weekly-updates/"
		}
	}
	// 规范化COS路径，确保以斜杠结尾
	if cosPath != "" && !strings.HasSuffix(cosPath, "/") {
		cosPath = cosPath + "/"
	}

	// 获取认证信息
	if secretID == "" {
		secretID = os.Getenv("WEEKLY_CLIENT_UPDATE_SECRET_ID")
	}
	if secretKey == "" {
		secretKey = os.Getenv("WEEKLY_CLIENT_UPDATE_SECRET_KEY")
	}

	if secretID == "" || secretKey == "" {
		log.Fatal("错误: 必须提供腾讯云认证信息（--secret-id/--secret-key 或设置 WEEKLY_CLIENT_UPDATE_SECRET_ID/WEEKLY_CLIENT_UPDATE_SECRET_KEY 环境变量）")
	}

	// 检查是否需要执行查询功能
	if listMode || historyClient != "" {
		if year == 0 || week == 0 {
			now := time.Now()
			year, week = now.ISOWeek()
		}

		if err := runQuery(bucket, region, secretID, secretKey, cosPath, year, week, listMode, historyClient); err != nil {
			log.Fatalf("查询失败: %v", err)
		}
		return
	}

	// 确定客户更新数据
	var clientUpdates []ClientUpdate

	if clientsJSON != "" {
		if err := json.Unmarshal([]byte(clientsJSON), &clientUpdates); err != nil {
			log.Fatalf("错误: 解析clients JSON失败: %v", err)
		}
	} else if clientName != "" && content != "" {
		clientUpdates = []ClientUpdate{
			{Name: clientName, Content: content},
		}
	} else {
		log.Fatal("错误: 必须提供客户更新信息（--client/--content 或 --clients）")
	}

	// 验证客户数据
	for i, update := range clientUpdates {
		if update.Name == "" {
			log.Fatalf("错误: 第%d个客户缺少名称", i+1)
		}
		if update.Content == "" {
			log.Fatalf("错误: 客户'%s'缺少更新内容", update.Name)
		}
	}

	// 确定年份和周数
	if year == 0 || week == 0 {
		now := time.Now()
		year, week = now.ISOWeek()
	}

	// 执行主逻辑
	if err := run(clientUpdates, bucket, region, secretID, secretKey, cosPath, year, week); err != nil {
		log.Fatalf("错误: %v", err)
	}
}

func run(clientUpdates []ClientUpdate, bucket, region, secretID, secretKey, cosPath string, year, week int) error {
	fmt.Printf("🚀 开始处理客户更新...\n")
	fmt.Printf("   周次: %d年第%02d周\n", year, week)
	fmt.Printf("   COS桶: %s (%s)\n", bucket, region)
	fmt.Printf("   COS路径: %s\n", cosPath)
	fmt.Printf("   客户数: %d\n\n", len(clientUpdates))

	// 初始化COS客户端
	fmt.Printf("🔗 连接腾讯云COS...\n")
	cosClient, err := NewRealCOSClient(bucket, region, secretID, secretKey)
	if err != nil {
		return fmt.Errorf("初始化COS客户端失败: %v", err)
	}
	fmt.Printf("   ✅ COS连接成功\n")

	// 初始化文件管理器（纯COS模式，无本地存储）
	fileManager := NewFileManager(cosClient)

	// 从COS加载周报
	fmt.Printf("\n📂 从COS加载周报文件...\n")
	report, err := fileManager.LoadWeeklyReport(year, week, cosPath)
	if err != nil {
		return fmt.Errorf("加载周报失败: %v", err)
	}

	initialClientCount := len(report.Clients)
	fmt.Printf("   当前已有客户: %d\n", initialClientCount)

	// 处理每个客户更新（原文照录，不做润色）
	fmt.Printf("\n🔄 处理客户更新（原文照录）...\n")
	for _, update := range clientUpdates {
		fmt.Printf("   👤 客户: %s\n", update.Name)

		if _, exists := report.Clients[update.Name]; exists {
			fmt.Printf("     ⚠️  客户已存在，将追加更新\n")
		} else {
			fmt.Printf("     ✅ 新客户\n")
		}

		// 原文照录，不做任何处理
		fileManager.ProcessClientUpdate(report, update)

		charCount := len([]rune(update.Content))
		fmt.Printf("     📝 原文记录: %d字\n", charCount)
	}

	// 保存周报（强制上传到COS）
	fmt.Printf("\n☁️  上传周报到COS...\n")
	if err := fileManager.SaveWeeklyReport(report, cosPath); err != nil {
		return fmt.Errorf("保存周报到COS失败: %v", err)
	}
	fmt.Printf("   ✅ COS上传成功\n")

	// 显示统计信息
	fmt.Printf("\n🎉 处理完成！\n")
	fmt.Printf("   📄 COS文件: %s\n", report.FilePath)
	fmt.Printf("   👥 总客户数: %d (+%d)\n", len(report.Clients), len(report.Clients)-initialClientCount)
	fmt.Printf("   📅 周次: %d年第%02d周\n", report.Year, report.Week)

	// 显示更新汇总
	clientNames := make([]string, 0, len(report.Clients))
	for name := range report.Clients {
		clientNames = append(clientNames, name)
	}
	for i := 0; i < len(clientNames)-1; i++ {
		for j := i + 1; j < len(clientNames); j++ {
			if clientNames[i] > clientNames[j] {
				clientNames[i], clientNames[j] = clientNames[j], clientNames[i]
			}
		}
	}

	fmt.Printf("\n📋 更新汇总:\n")
	for _, update := range clientUpdates {
		if content, exists := report.Clients[update.Name]; exists {
			fmt.Printf("\n## %s\n", update.Name)
			charCount := len([]rune(content))
			preview := content
			if charCount > 100 {
				runes := []rune(content)
				preview = string(runes[:100]) + "..."
			}
			fmt.Printf("%s\n", preview)
			fmt.Printf("   （共%d字）\n", charCount)
		}
	}

	return nil
}

// runQuery 执行查询功能（列表或历史查询）
func runQuery(bucket, region, secretID, secretKey, cosPath string, year, week int, listMode bool, historyClient string) error {
	if cosPath != "" && !strings.HasSuffix(cosPath, "/") {
		cosPath = cosPath + "/"
	}

	fmt.Printf("🔍 开始执行查询...\n")
	fmt.Printf("   COS桶: %s (%s)\n", bucket, region)
	fmt.Printf("   COS路径: %s\n", cosPath)

	cosClient, err := NewRealCOSClient(bucket, region, secretID, secretKey)
	if err != nil {
		return fmt.Errorf("初始化COS客户端失败: %v", err)
	}
	fmt.Printf("   ✅ COS连接成功\n")

	fileManager := NewFileManager(cosClient)

	if listMode {
		return listWeeklyReport(fileManager, cosPath, year, week)
	} else if historyClient != "" {
		return queryClientHistory(fileManager, cosPath, historyClient)
	}

	return fmt.Errorf("未指定查询模式")
}

// listWeeklyReport 列出周报客户情况
func listWeeklyReport(fm *FileManager, cosPath string, year, week int) error {
	fmt.Printf("\n📊 查询客户汇总...\n")
	fmt.Printf("   周次: %d年第%02d周\n\n", year, week)

	report, err := fm.LoadWeeklyReport(year, week, cosPath)
	if err != nil {
		return fmt.Errorf("加载周报失败: %v", err)
	}

	if len(report.Clients) == 0 {
		fmt.Printf("   ℹ️  本周暂无客户更新\n")
		return nil
	}

	clientNames := make([]string, 0, len(report.Clients))
	for name := range report.Clients {
		clientNames = append(clientNames, name)
	}
	for i := 0; i < len(clientNames)-1; i++ {
		for j := i + 1; j < len(clientNames); j++ {
			if clientNames[i] > clientNames[j] {
				clientNames[i], clientNames[j] = clientNames[j], clientNames[i]
			}
		}
	}

	fmt.Printf("   👥 本周共 %d 个客户:\n\n", len(clientNames))

	totalChars := 0
	for i, name := range clientNames {
		content := report.Clients[name]
		charCount := len([]rune(content))
		totalChars += charCount

		summary := content
		if charCount > 100 {
			runes := []rune(content)
			summary = string(runes[:100]) + "..."
		}

		fmt.Printf("   %d. %s (%d字)\n", i+1, name, charCount)
		fmt.Printf("      %s\n", summary)
		fmt.Println()
	}

	fmt.Printf("📈 统计: %d个客户，总字数约%d字\n", len(clientNames), totalChars)
	return nil
}

// queryClientHistory 查询客户历史更新
func queryClientHistory(fm *FileManager, cosPath, clientName string) error {
	fmt.Printf("\n📜 查询客户历史更新...\n")
	fmt.Printf("   客户: %s\n\n", clientName)

	fmt.Printf("🔍 扫描COS文件...\n")
	files, err := fm.cosClient.ListFiles(context.Background(), cosPath)
	if err != nil {
		return fmt.Errorf("扫描COS文件失败: %v", err)
	}

	var weeklyFiles []string
	for _, file := range files {
		if strings.Contains(file, "/week-") && strings.HasSuffix(file, ".md") {
			weeklyFiles = append(weeklyFiles, file)
		}
	}

	if len(weeklyFiles) == 0 {
		fmt.Printf("   ℹ️  未找到周报文件\n")
		return nil
	}

	fmt.Printf("   找到 %d 个周报文件\n", len(weeklyFiles))

	sortWeeklyFiles(weeklyFiles)

	var foundWeeks []string
	clientHistory := make(map[string]string)

	for _, file := range weeklyFiles {
		fileYear, fileWeek, err := parseYearWeekFromPath(file)
		if err != nil {
			continue
		}

		report, err := fm.LoadWeeklyReport(fileYear, fileWeek, cosPath)
		if err != nil {
			continue
		}

		if content, exists := report.Clients[clientName]; exists {
			weekKey := fmt.Sprintf("%d年第%02d周", fileYear, fileWeek)
			foundWeeks = append(foundWeeks, weekKey)
			clientHistory[weekKey] = content
		}
	}

	if len(foundWeeks) == 0 {
		fmt.Printf("\n   ℹ️  未找到客户 '%s' 的历史更新记录\n", clientName)
		return nil
	}

	fmt.Printf("\n✅ 找到客户 '%s' 的 %d 次历史更新:\n\n", clientName, len(foundWeeks))

	for i, weekKey := range foundWeeks {
		content := clientHistory[weekKey]
		charCount := len([]rune(content))

		summary := content
		if charCount > 120 {
			runes := []rune(content)
			summary = string(runes[:120]) + "..."
		}

		fmt.Printf("   📅 %s (%d字)\n", weekKey, charCount)
		fmt.Printf("      %s\n", summary)

		if i < len(foundWeeks)-1 {
			fmt.Println()
		}
	}

	fmt.Printf("\n📈 统计: 共 %d 次历史更新\n", len(foundWeeks))
	return nil
}

// sortWeeklyFiles 按年份和周数排序周报文件（从新到旧）
func sortWeeklyFiles(files []string) {
	sort.Slice(files, func(i, j int) bool {
		yearI, weekI, errI := parseYearWeekFromPath(files[i])
		yearJ, weekJ, errJ := parseYearWeekFromPath(files[j])

		if errI != nil || errJ != nil {
			return errI == nil && errJ != nil
		}

		if yearI != yearJ {
			return yearI > yearJ
		}
		return weekI > weekJ
	})
}

// parseYearWeekFromPath 从文件路径解析年份和周数
// 路径格式: weekly-updates/YYYY/week-WW.md
func parseYearWeekFromPath(path string) (year, week int, err error) {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return 0, 0, fmt.Errorf("无效的文件路径格式: %s", path)
	}

	yearStr := parts[len(parts)-2]
	year, err = strconv.Atoi(yearStr)
	if err != nil {
		return 0, 0, fmt.Errorf("解析年份失败: %s (%v)", yearStr, err)
	}

	filename := parts[len(parts)-1]
	filename = strings.TrimSuffix(filename, ".md")

	if !strings.HasPrefix(filename, "week-") {
		return 0, 0, fmt.Errorf("无效的文件名格式: %s", filename)
	}

	weekStr := strings.TrimPrefix(filename, "week-")
	week, err = strconv.Atoi(weekStr)
	if err != nil {
		return 0, 0, fmt.Errorf("解析周数失败: %s (%v)", weekStr, err)
	}

	return year, week, nil
}
