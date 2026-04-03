package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// 程序版本
const Version = "1.0.0"

func main() {
	// 解析命令行参数
	var (
		clientName  string
		content     string
		clientsJSON string
		bucket      string
		region      string
		secretID    string
		secretKey   string
		cosPath     string
		year        int
		week        int
		showHelp    bool
		showVersion bool
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
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "每周客户更新管理工具 v%s\n\n", Version)
		fmt.Fprintf(os.Stderr, "使用方法：\n")
		fmt.Fprintf(os.Stderr, "  %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项：\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例：\n")
		fmt.Fprintf(os.Stderr, "  单个客户更新：\n")
		fmt.Fprintf(os.Stderr, "    %s --client \"好未来\" --content \"项目进展...\" --bucket my-bucket\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  多个客户更新：\n")
		fmt.Fprintf(os.Stderr, "    %s --clients '[{\"name\":\"客户A\",\"content\":\"内容A\"}]' --bucket my-bucket\n", os.Args[0])
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
			region = "ap-beijing" // 默认北京区域
		}
	}
	
	// 设置COS上传路径
	if cosPath == "" {
		cosPath = os.Getenv("WEEKLY_CLIENT_UPDATE_COS_PATH")
		if cosPath == "" {
			cosPath = "weekly-updates/" // 默认路径
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
	
	// 确定客户更新数据
	var clientUpdates []ClientUpdate
	
	if clientsJSON != "" {
		// 解析JSON格式的多个客户
		if err := json.Unmarshal([]byte(clientsJSON), &clientUpdates); err != nil {
			log.Fatalf("错误: 解析clients JSON失败: %v", err)
		}
	} else if clientName != "" && content != "" {
		// 单个客户
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
	// 规范化COS路径，确保以斜杠结尾
	if cosPath != "" && !strings.HasSuffix(cosPath, "/") {
		cosPath = cosPath + "/"
	}
	
	fmt.Printf("🚀 开始处理客户更新...\n")
	fmt.Printf("   周次: %d年第%02d周\n", year, week)
	fmt.Printf("   COS桶: %s (%s)\n", bucket, region)
	fmt.Printf("   COS路径: %s\n", cosPath)
	fmt.Printf("   客户数: %d\n\n", len(clientUpdates))
	
	// 初始化文本润色器
	polisher := &EnhancedPolisher{}
	
	// 初始化COS客户端
	fmt.Printf("🔗 连接腾讯云COS...\n")
	cosClient, err := NewRealCOSClient(bucket, region, secretID, secretKey)
	if err != nil {
		return fmt.Errorf("初始化COS客户端失败: %v", err)
	}
	fmt.Printf("   ✅ COS连接成功\n")
	
	// 初始化文件管理器
	localDir := "./weekly-updates"
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("创建本地目录失败: %v", err)
	}
	
	fileManager := &FileManager{
		localDir:  localDir,
		cosClient: cosClient,
		polisher:  polisher,
	}
	
	// 加载周报
	fmt.Printf("\n📂 加载周报文件...\n")
	report, err := fileManager.LoadWeeklyReport(year, week, cosPath)
	if err != nil {
		return fmt.Errorf("加载周报失败: %v", err)
	}
	
	initialClientCount := len(report.Clients)
	fmt.Printf("   当前已有客户: %d\n", initialClientCount)
	
	// 处理每个客户更新
	fmt.Printf("\n🔄 处理客户更新...\n")
	for _, update := range clientUpdates {
		fmt.Printf("   👤 客户: %s\n", update.Name)
		
		// 检查是否已存在
		if _, exists := report.Clients[update.Name]; exists {
			fmt.Printf("     ⚠️  客户已存在，将追加更新\n")
		} else {
			fmt.Printf("     ✅ 新客户\n")
		}
		
		// 处理更新
		if err := fileManager.ProcessClientUpdate(report, update); err != nil {
			fmt.Printf("     ❌ 处理失败: %v\n", err)
			continue
		}
		
		// 显示润色后字数
		if polished, exists := report.Clients[update.Name]; exists {
			charCount := len([]rune(polished))
			fmt.Printf("     📝 润色完成: %d字\n", charCount)
		}
	}
	
	// 保存周报
	fmt.Printf("\n💾 保存周报...\n")
	if err := fileManager.SaveWeeklyReport(report, cosPath); err != nil {
		return fmt.Errorf("保存周报失败: %v", err)
	}
	
	// 显示统计信息
	fmt.Printf("\n🎉 处理完成！\n")
	fmt.Printf("   📄 文件: %s\n", report.FilePath)
	fmt.Printf("   👥 总客户数: %d (+%d)\n", len(report.Clients), len(report.Clients)-initialClientCount)
	fmt.Printf("   📅 周次: %d年第%02d周\n", report.Year, report.Week)
	
	// 显示新添加/更新的客户（带序号，与文件顺序一致）
	// 生成排序的客户列表（与文件保存顺序一致）
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
	// 创建客户名到序号的映射
	clientIndex := make(map[string]int)
	for i, name := range clientNames {
		clientIndex[name] = i + 1
	}
	
	fmt.Printf("\n📋 更新汇总:\n")
	for _, update := range clientUpdates {
		if polished, exists := report.Clients[update.Name]; exists {
			index := clientIndex[update.Name]
			fmt.Printf("\n## %d. %s\n", index, update.Name)
			
			// 提取最近添加的内容
			paragraphs := splitParagraphs(polished)
			if len(paragraphs) > 0 {
				lastParagraph := paragraphs[len(paragraphs)-1]
				charCount := len([]rune(lastParagraph))
				
				// 显示最后一段的前100字
				preview := lastParagraph
				if charCount > 100 {
					runes := []rune(lastParagraph)
					preview = string(runes[:100]) + "..."
				}
				fmt.Printf("%s\n", preview)
				fmt.Printf("   （本段%d字）\n", charCount)
			}
		}
	}
	
	// 提醒COS上传
	fmt.Printf("\n☁️  文件已同步到腾讯云COS:\n")
	fmt.Printf("   bucket: %s\n", bucket)
	fmt.Printf("   key: %s%d/week-%02d.md\n", cosPath, year, week)
	
	return nil
}

// splitParagraphs 将文本分割成段落
func splitParagraphs(text string) []string {
	var paragraphs []string
	var current strings.Builder
	
	for _, r := range text {
		if r == '\n' {
			if current.Len() > 0 {
				paragraphs = append(paragraphs, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}
	
	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}
	
	return paragraphs
}

// EnhancedPolisher 增强的文本润色器
type EnhancedPolisher struct{}

func (p *EnhancedPolisher) Polish(text string, targetLength int) (string, error) {
	// 基础清理
	text = strings.TrimSpace(text)
	
	// 替换常见的非正式表达
	text = p.replaceInformalExpressions(text)
	
	// 优化句子结构
	text = p.optimizeSentenceStructure(text)
	
	// 控制字数
	text = p.adjustLength(text, targetLength)
	
	// 最后整理
	text = p.finalPolish(text)
	
	return text, nil
}

func (p *EnhancedPolisher) replaceInformalExpressions(text string) string {
	replacements := map[string]string{
		"搞定了": "已完成",
		"弄好了": "已处理完成", 
		"应该可以": "预计可行",
		"大概": "大约",
		"好像": "似乎",
		"挺": "相当",
		"有点": "略微",
		"很多": "大量",
		"很快": "迅速",
		"马上": "立即",
		"还行": "尚可",
		"不错": "良好",
		"特别好": "非常优秀",
		"贼好": "极其出色",
		"巨": "非常",
		"超": "极其",
		"贼": "极其",
	}
	
	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	
	return result
}

func (p *EnhancedPolisher) optimizeSentenceStructure(text string) string {
	// 分割句子
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return text
	}
	
	// 优化每个句子
	optimized := make([]string, len(sentences))
	for i, sentence := range sentences {
		optimized[i] = p.optimizeSingleSentence(sentence)
	}
	
	// 重新组合
	return strings.Join(optimized, " ")
}

func (p *EnhancedPolisher) optimizeSingleSentence(sentence string) string {
	sentence = strings.TrimSpace(sentence)
	if sentence == "" {
		return ""
	}
	
	// 确保句子有主语
	if !p.hasSubject(sentence) {
		// 尝试添加隐含主语
		sentence = "客户" + sentence
	}
	
	// 优化连接词
	sentence = p.optimizeConnectors(sentence)
	
	return sentence
}

func (p *EnhancedPolisher) hasSubject(sentence string) bool {
	// 简单的主题检查（实际应用中可以更复杂）
	subjectIndicators := []string{"客户", "项目", "团队", "我们", "他们", "该"}
	for _, indicator := range subjectIndicators {
		if strings.Contains(sentence, indicator) {
			return true
		}
	}
	return false
}

func (p *EnhancedPolisher) optimizeConnectors(sentence string) string {
	// 优化句子开头的连接词
	connectors := map[string]string{
		"然后": "随后",
		"接着": "接下来",
		"另外": "此外",
		"还有": "同时",
		"但是": "然而",
		"可是": "不过",
	}
	
	for old, new := range connectors {
		if strings.HasPrefix(sentence, old) {
			sentence = new + sentence[len(old):]
			break
		}
	}
	
	return sentence
}

func (p *EnhancedPolisher) adjustLength(text string, targetLength int) string {
	currentLength := len([]rune(text))
	
	// 如果长度合适，直接返回
	minLength := int(float64(targetLength) * 0.8)
	maxLength := int(float64(targetLength) * 1.2)
	if currentLength >= minLength && currentLength <= maxLength {
		return text
	}
	
	// 如果太短，适当扩展
	if currentLength < minLength {
		return p.expandText(text, targetLength)
	}
	
	// 如果太长，适当精简
	return p.shrinkText(text, targetLength)
}

func (p *EnhancedPolisher) expandText(text string, targetLength int) string {
	// 分析文本内容
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return text
	}
	
	// 添加细节描述
	enhanced := text
	
	// 添加一些商务场景常见的补充
	enhancements := []string{
		"从整体进展来看，",
		"值得注意的是，",
		"具体而言，",
		"在此背景下，",
		"针对这一情况，",
	}
	
	// 在适当位置添加增强表达
	for _, enhancement := range enhancements {
		if len([]rune(enhanced)) < targetLength {
			// 在第一个句号后添加
			if strings.Contains(enhanced, "。") {
				parts := strings.SplitN(enhanced, "。", 2)
				if len(parts) == 2 {
					enhanced = parts[0] + "。" + enhancement + parts[1]
				}
			}
		}
	}
	
	// 如果仍然太短，添加总结性语句
	minThreshold := int(float64(targetLength) * 0.9)
	if len([]rune(enhanced)) < minThreshold {
		conclusions := []string{
			"总体而言，项目进展顺利。",
			"后续将持续关注相关进展。",
			"预计下一步将按计划推进。",
		}
		
		for _, conclusion := range conclusions {
			if len([]rune(enhanced)) < targetLength {
				enhanced += " " + conclusion
			}
		}
	}
	
	return enhanced
}

func (p *EnhancedPolisher) shrinkText(text string, targetLength int) string {
	sentences := splitSentences(text)
	if len(sentences) <= 1 {
		// 只有一个句子，直接截取
		return p.truncateText(text, targetLength)
	}
	
	// 尝试保留核心句子
	coreSentences := []string{}
	currentLength := 0
	
	// 优先保留包含关键词的句子
	keywords := []string{"完成", "进展", "成功", "重要", "关键", "下一步", "计划"}
	
	for _, sentence := range sentences {
		sentenceLength := len([]rune(sentence))
		
		// 检查是否包含关键词
		hasKeyword := false
		for _, keyword := range keywords {
			if strings.Contains(sentence, keyword) {
				hasKeyword = true
				break
			}
		}
		
		// 如果包含关键词或长度允许，保留该句子
		if hasKeyword || currentLength+sentenceLength <= targetLength {
			coreSentences = append(coreSentences, sentence)
			currentLength += sentenceLength + 1 // +1 for space
		}
		
		if currentLength >= targetLength {
			break
		}
	}
	
	result := strings.Join(coreSentences, " ")
	
	// 如果仍然太长，进行截取
	if len([]rune(result)) > targetLength {
		result = p.truncateText(result, targetLength)
	}
	
	return result
}

func (p *EnhancedPolisher) truncateText(text string, targetLength int) string {
	runes := []rune(text)
	if len(runes) <= targetLength {
		return text
	}
	
	// 寻找合适的截断点（句子边界或标点）
	truncateAt := targetLength
	for i := targetLength; i > targetLength-20 && i > 0; i-- {
		if i < len(runes) {
			r := runes[i]
			if r == '。' || r == '；' || r == '，' || r == '.' || r == ';' || r == ',' {
				truncateAt = i + 1
				break
			}
		}
	}
	
	if truncateAt < len(runes) {
		return string(runes[:truncateAt]) + "..."
	}
	return string(runes[:targetLength]) + "..."
}

func (p *EnhancedPolisher) finalPolish(text string) string {
	// 确保以句号结束
	text = strings.TrimSpace(text)
	if text != "" && !strings.HasSuffix(text, "。") && !strings.HasSuffix(text, ".") {
		text += "。"
	}
	
	// 标准化空格
	text = strings.Join(strings.Fields(text), " ")
	
	return text
}

// splitSentences 将文本分割成句子
func splitSentences(text string) []string {
	// 简单的句子分割（实际应用中可以更复杂）
	var sentences []string
	var current strings.Builder
	
	for _, r := range text {
		current.WriteRune(r)
		if r == '。' || r == '.' || r == '；' || r == ';' {
			sentence := strings.TrimSpace(current.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			current.Reset()
		}
	}
	
	// 处理最后一个句子
	lastSentence := strings.TrimSpace(current.String())
	if lastSentence != "" {
		sentences = append(sentences, lastSentence)
	}
	
	return sentences
}