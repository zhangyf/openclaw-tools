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
const Version = "1.0.0"

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
	
	// 检查是否需要执行查询功能
	if listMode || historyClient != "" {
		// 确定年份和周数
		if year == 0 || week == 0 {
			now := time.Now()
			year, week = now.ISOWeek()
		}
		
		// 执行查询功能
		if err := runQuery(bucket, region, secretID, secretKey, cosPath, year, week, listMode, historyClient); err != nil {
			log.Fatalf("查询失败: %v", err)
		}
		return
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
	
	enhanced := text
	currentLength := len([]rune(enhanced))
	
	// 轻微润色策略：只做最基本扩展，避免任何重复
	// 如果文本太短，只添加1-2句最相关的扩展
	
	if currentLength < targetLength {
		// 分析文本内容，选择最相关的扩展方向
		hasDataSize := strings.Contains(enhanced, "TB") || strings.Contains(enhanced, "GB") || strings.Contains(enhanced, "数据量")
		hasDataTransfer := strings.Contains(enhanced, "传输") || strings.Contains(enhanced, "迁移") || strings.Contains(enhanced, "复制")
		hasTechSolution := strings.Contains(enhanced, "方案") || strings.Contains(enhanced, "建议") || strings.Contains(enhanced, "使用")
		hasDetection := strings.Contains(enhanced, "检测") || strings.Contains(enhanced, "审核") || strings.Contains(enhanced, "扫描")
		
		// 根据内容优先级选择扩展句（只选一个最相关的）
		var extension string
		
		// 优先级1：数据迁移 + 数据量（最具体）
		if hasDataTransfer && hasDataSize {
			extension = "该数据迁移需要考虑传输效率和安全性。"
		} else if hasDataSize {
			// 优先级2：数据量相关
			extension = "该数据规模对存储和传输都有一定要求。"
		} else if hasDataTransfer {
			// 优先级3：数据传输相关
			extension = "数据传输需要考虑安全性和效率。"
		} else if hasTechSolution {
			// 优先级4：技术方案相关
			extension = "该方案需要进一步评估可行性和实施细节。"
		} else if hasDetection {
			// 优先级5：检测审核相关
			extension = "检测机制需要平衡准确性和处理效率。"
		} else {
			// 通用扩展（最后选择）
			extension = "项目需要进一步细化和评估实施方案。"
		}
		
		// 检查扩展句是否与原文重复
		if extension != "" {
			// 检查扩展句的关键词是否已存在于原文中
			extensionKeywords := []string{}
			if strings.Contains(extension, "数据迁移") {
				extensionKeywords = append(extensionKeywords, "数据迁移", "传输", "迁移")
			}
			if strings.Contains(extension, "数据规模") {
				extensionKeywords = append(extensionKeywords, "数据规模", "数据量", "存储")
			}
			if strings.Contains(extension, "数据传输") {
				extensionKeywords = append(extensionKeywords, "数据传输", "传输", "安全")
			}
			if strings.Contains(extension, "方案") {
				extensionKeywords = append(extensionKeywords, "方案", "评估", "实施")
			}
			if strings.Contains(extension, "检测机制") {
				extensionKeywords = append(extensionKeywords, "检测", "机制", "准确")
			}
			
			// 如果扩展句的关键词大多已存在，则不添加
			existingKeywords := 0
			for _, keyword := range extensionKeywords {
				if strings.Contains(enhanced, keyword) {
					existingKeywords++
				}
			}
			
			// 如果超过一半的关键词已存在，或者扩展句与任何句子高度相似，则不添加
			shouldAdd := true
			if existingKeywords > len(extensionKeywords)/2 {
				shouldAdd = false
			}
			
			// 检查扩展句是否与原文中的任何句子相似
			for _, sentence := range sentences {
				if strings.Contains(sentence, extension) || strings.Contains(extension, sentence) {
					shouldAdd = false
					break
				}
				
				// 检查句子长度和内容相似度
				if len(sentence) > 10 && len(extension) > 10 {
					// 简单的相似度检查：如果两个句子有超过60%的相同词汇，视为重复
					words1 := strings.Fields(sentence)
					words2 := strings.Fields(extension)
					commonWords := 0
					
					for _, w1 := range words1 {
						for _, w2 := range words2 {
							if w1 == w2 && len(w1) > 1 {
								commonWords++
								break
							}
						}
					}
					
					if commonWords > len(words1)/2 || commonWords > len(words2)/2 {
						shouldAdd = false
						break
					}
				}
			}
			
			if shouldAdd {
				enhanced = enhanced + " " + extension
				currentLength = len([]rune(enhanced))
			}
		}
		
		// 如果添加了一个扩展后仍然太短，可以考虑添加第二个扩展（但更谨慎）
		if currentLength < int(float64(targetLength)*0.8) {
			// 选择第二个扩展方向（与第一个不同）
			var secondExtension string
			
			if hasDataTransfer && !strings.Contains(enhanced, "效率") && !strings.Contains(enhanced, "安全") {
				secondExtension = "需要关注传输过程中的数据一致性。"
			} else if hasTechSolution && !strings.Contains(enhanced, "评估") {
				secondExtension = "建议进一步细化技术实现方案。"
			} else if hasDetection && !strings.Contains(enhanced, "平衡") {
				secondExtension = "检测准确率是重要的性能指标。"
			}
			
			// 检查第二个扩展是否应该添加
			if secondExtension != "" {
				shouldAddSecond := true
				for _, sentence := range sentences {
					if strings.Contains(sentence, secondExtension) {
						shouldAddSecond = false
						break
					}
				}
				
				if shouldAddSecond && !strings.Contains(enhanced, secondExtension) {
					enhanced = enhanced + " " + secondExtension
				}
			}
		}
	}
	
	// 清理多余的句号和空格
	enhanced = strings.TrimSpace(enhanced)
	enhanced = strings.ReplaceAll(enhanced, "。。", "。")
	enhanced = strings.ReplaceAll(enhanced, "  ", " ")
	
	// 确保以句号结束
	if enhanced != "" && !strings.HasSuffix(enhanced, "。") {
		enhanced = enhanced + "。"
	}
	
	return enhanced
}

// restructureWithDetails 重新组织文本结构，增加细节描述
func (p *EnhancedPolisher) restructureWithDetails(text string, targetLength int) string {
	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return text
	}
	
	// 分析句子内容，尝试添加更多细节
	enhancedSentences := make([]string, 0, len(sentences)*2)
	
	for _, sentence := range sentences {
		enhancedSentences = append(enhancedSentences, sentence)
		
		// 基于句子内容添加相关细节
		if strings.Contains(sentence, "数据") && strings.Contains(sentence, "TB") {
			enhancedSentences = append(enhancedSentences, "该数据规模对存储和传输都提出了较高要求。")
		}
		
		if strings.Contains(sentence, "迁移") || strings.Contains(sentence, "传输") {
			enhancedSentences = append(enhancedSentences, "迁移过程需要确保数据的一致性和完整性。")
		}
		
		if strings.Contains(sentence, "方案") || strings.Contains(sentence, "建议") {
			enhancedSentences = append(enhancedSentences, "该方案需要综合考虑技术可行性和实施成本。")
		}
		
		if strings.Contains(sentence, "检测") || strings.Contains(sentence, "审核") {
			enhancedSentences = append(enhancedSentences, "检测机制需要平衡准确性和处理效率。")
		}
	}
	
	// 重新组合句子
	result := strings.Join(enhancedSentences, " ")
	
	// 如果还是太短，适当重复核心信息
	if len([]rune(result)) < targetLength && len(sentences) > 0 {
		// 提取核心句子并重新表述
		coreSentence := sentences[0]
		rephrased := p.rephraseSentence(coreSentence)
		if rephrased != "" && !strings.Contains(result, rephrased) {
			result = rephrased + " " + result
		}
	}
	
	return result
}

// rephraseSentence 重新表述句子，避免重复
func (p *EnhancedPolisher) rephraseSentence(sentence string) string {
	if len(sentence) == 0 {
		return ""
	}
	
	// 简单的同义词替换和句式变换
	rephrased := sentence
	
	// 替换部分词汇
	replacements := map[string]string{
		"需要": "要求",
		"建议": "推荐",
		"使用": "采用",
		"传输": "迁移",
		"数据": "信息",
		"客户": "用户",
		"项目": "任务",
	}
	
	for old, new := range replacements {
		if strings.Contains(rephrased, old) && !strings.Contains(rephrased, new) {
			// 只替换一次，避免过度替换
			rephrased = strings.Replace(rephrased, old, new, 1)
			break
		}
	}
	
	// 改变句式
	if strings.HasPrefix(rephrased, "客户") {
		rephrased = strings.TrimPrefix(rephrased, "客户")
		rephrased = "用户需求方面，" + rephrased
	}
	
	if strings.Contains(rephrased, "已经") {
		rephrased = strings.Replace(rephrased, "已经", "目前", -1)
	}
	
	return rephrased
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

// runQuery 执行查询功能（列表或历史查询）
func runQuery(bucket, region, secretID, secretKey, cosPath string, year, week int, listMode bool, historyClient string) error {
	// 规范化COS路径，确保以斜杠结尾
	if cosPath != "" && !strings.HasSuffix(cosPath, "/") {
		cosPath = cosPath + "/"
	}
	
	fmt.Printf("🔍 开始执行查询...\n")
	fmt.Printf("   COS桶: %s (%s)\n", bucket, region)
	fmt.Printf("   COS路径: %s\n", cosPath)
	
	// 初始化文本润色器（用于历史查询时可能需要）
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
	
	// 根据查询模式执行不同的操作
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
	
	// 加载周报
	report, err := fm.LoadWeeklyReport(year, week, cosPath)
	if err != nil {
		return fmt.Errorf("加载周报失败: %v", err)
	}
	
	if len(report.Clients) == 0 {
		fmt.Printf("   ℹ️  本周暂无客户更新\n")
		return nil
	}
	
	// 生成排序的客户列表（按字母顺序）
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
	
	// 显示客户列表
	fmt.Printf("   👥 本周共 %d 个客户:\n\n", len(clientNames))
	
	totalChars := 0
	for i, name := range clientNames {
		content := report.Clients[name]
		charCount := len([]rune(content))
		totalChars += charCount
		
		// 截取前100个字符作为摘要
		summary := content
		if charCount > 100 {
			runes := []rune(content)
			summary = string(runes[:100]) + "..."
		}
		
		fmt.Printf("   %d. %s (%d字)\n", i+1, name, charCount)
		fmt.Printf("      %s\n", summary)
		
		// 显示原始记录数量（如果有）
		if rawData, exists := report.RawData[name]; exists && len(rawData) > 0 {
			fmt.Printf("      📝 原始记录: %d 条\n", len(rawData))
		}
		
		fmt.Println()
	}
	
	fmt.Printf("📈 统计: %d个客户，总字数约%d字\n", len(clientNames), totalChars)
	return nil
}

// queryClientHistory 查询客户历史更新
func queryClientHistory(fm *FileManager, cosPath, clientName string) error {
	fmt.Printf("\n📜 查询客户历史更新...\n")
	fmt.Printf("   客户: %s\n\n", clientName)
	
	// 获取COS中所有周报文件
	fmt.Printf("🔍 扫描COS文件...\n")
	files, err := fm.cosClient.ListFiles(context.Background(), cosPath)
	if err != nil {
		return fmt.Errorf("扫描COS文件失败: %v", err)
	}
	
	// 过滤出周报文件（格式: weekly-updates/YYYY/week-WW.md）
	var weeklyFiles []string
	for _, file := range files {
		// 检查文件路径格式
		if strings.Contains(file, "/week-") && strings.HasSuffix(file, ".md") {
			weeklyFiles = append(weeklyFiles, file)
		}
	}
	
	if len(weeklyFiles) == 0 {
		fmt.Printf("   ℹ️  未找到周报文件\n")
		return nil
	}
	
	fmt.Printf("   找到 %d 个周报文件\n", len(weeklyFiles))
	
	// 按年份和周数排序文件（从新到旧）
	sortWeeklyFiles(weeklyFiles)
	
	// 逐个检查文件是否包含该客户
	var foundWeeks []string
	clientHistory := make(map[string]string) // week -> content
	totalFound := 0
	
	for _, file := range weeklyFiles {
		// 从文件路径解析年份和周数
		year, week, err := parseYearWeekFromPath(file)
		if err != nil {
			fmt.Printf("   ⚠️  无法解析文件路径: %s (%v)\n", file, err)
			continue
		}
		
		// 加载周报
		report, err := fm.LoadWeeklyReport(year, week, cosPath)
		if err != nil {
			fmt.Printf("   ⚠️  加载周报失败: %s (%v)\n", file, err)
			continue
		}
		
		// 检查是否包含该客户
		if content, exists := report.Clients[clientName]; exists {
			weekKey := fmt.Sprintf("%d年第%02d周", year, week)
			foundWeeks = append(foundWeeks, weekKey)
			clientHistory[weekKey] = content
			totalFound++
		}
		
		// 显示进度（每10个文件显示一次）
		if len(foundWeeks)%10 == 0 {
			fmt.Printf("   📊 已扫描 %d 个文件，找到 %d 次更新\n", totalFound, len(foundWeeks))
		}
	}
	
	if len(foundWeeks) == 0 {
		fmt.Printf("\n   ℹ️  未找到客户 '%s' 的历史更新记录\n", clientName)
		return nil
	}
	
	// 显示历史更新
	fmt.Printf("\n✅ 找到客户 '%s' 的 %d 次历史更新:\n\n", clientName, len(foundWeeks))
	
	for i, weekKey := range foundWeeks {
		content := clientHistory[weekKey]
		charCount := len([]rune(content))
		
		// 截取前120个字符作为摘要
		summary := content
		if charCount > 120 {
			runes := []rune(content)
			summary = string(runes[:120]) + "..."
		}
		
		fmt.Printf("   📅 %s (%d字)\n", weekKey, charCount)
		fmt.Printf("      %s\n", summary)
		
		// 如果有原始记录，显示数量
		// 注意：这里需要重新加载原始记录，因为report变量已被重用
		// 为了简化，暂时不显示原始记录数量
		
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
		// 从新到旧排序
		yearI, weekI, errI := parseYearWeekFromPath(files[i])
		yearJ, weekJ, errJ := parseYearWeekFromPath(files[j])
		
		// 如果解析失败，放在最后
		if errI != nil || errJ != nil {
			return errI == nil && errJ != nil
		}
		
		// 先按年份降序，再按周数降序
		if yearI != yearJ {
			return yearI > yearJ
		}
		return weekI > weekJ
	})
}

// parseYearWeekFromPath 从文件路径解析年份和周数
// 路径格式: weekly-updates/YYYY/week-WW.md
func parseYearWeekFromPath(path string) (year, week int, err error) {
	// 分割路径
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return 0, 0, fmt.Errorf("无效的文件路径格式: %s", path)
	}
	
	// 获取年份部分（parts[-2]）
	yearStr := parts[len(parts)-2]
	year, err = strconv.Atoi(yearStr)
	if err != nil {
		return 0, 0, fmt.Errorf("解析年份失败: %s (%v)", yearStr, err)
	}
	
	// 获取文件名部分（parts[-1]）
	filename := parts[len(parts)-1]
	// 移除扩展名
	filename = strings.TrimSuffix(filename, ".md")
	
	// 解析周数（格式: week-WW）
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