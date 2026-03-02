#!/usr/bin/env node
/**
 * 于田称呼检查脚本
 * 确保在群聊中正确称呼"于老师"
 */

const fs = require('fs');
const path = require('path');

// 加载记忆
const memoryPath = '/home/zhangyufeng/.openclaw/workspace/MEMORY.md';
const memory = fs.readFileSync(memoryPath, 'utf8');

// 提取于田信息
function extractYutianInfo() {
    const lines = memory.split('\n');
    let inYutianSection = false;
    const info = {
        correctName: '于田',
        correctTitle: '于老师',
        wrongTitles: ['田女士', '田小姐', '田'],
        notes: []
    };
    
    for (const line of lines) {
        if (line.includes('### 于田')) {
            inYutianSection = true;
            continue;
        }
        if (inYutianSection && line.startsWith('### ')) {
            break;
        }
        if (inYutianSection) {
            if (line.includes('当前称呼')) {
                const match = line.match(/当前称呼[：:]\s*(.+?)(?:【|$)/);
                if (match) {
                    info.correctTitle = match[1].trim();
                }
            }
            if (line.includes('错误记录')) {
                info.notes.push(line.trim());
            }
        }
    }
    
    return info;
}

// 检查文本中的称呼
function checkTextForWrongTitles(text, yutianInfo) {
    const wrongFound = [];
    
    for (const wrongTitle of yutianInfo.wrongTitles) {
        if (text.includes(wrongTitle)) {
            wrongFound.push(wrongTitle);
        }
    }
    
    return wrongFound;
}

// 生成正确称呼提示
function generateCorrection(wrongTitles, yutianInfo) {
    if (wrongTitles.length === 0) {
        return null;
    }
    
    let correction = `⚠️ 称呼错误检测:\n`;
    correction += `错误称呼: ${wrongTitles.join(', ')}\n`;
    correction += `正确称呼: ${yutianInfo.correctTitle}\n`;
    correction += `记忆记录: ${yutianInfo.notes.join('; ')}\n`;
    
    return correction;
}

// 主函数
function main() {
    console.log('🔍 检查于田称呼配置...');
    
    const yutianInfo = extractYutianInfo();
    console.log(`正确姓名: ${yutianInfo.correctName}`);
    console.log(`正确称呼: ${yutianInfo.correctTitle}`);
    console.log(`错误称呼: ${yutianInfo.wrongTitles.join(', ')}`);
    
    // 测试文本
    const testTexts = [
        '田女士最近怎么样？',
        '于老师的孩子应该放假了吧',
        '田小姐在美国还好吗？',
        '于田最近有联系吗？'
    ];
    
    console.log('\n📝 测试检查:');
    testTexts.forEach((text, i) => {
        const wrongTitles = checkTextForWrongTitles(text, yutianInfo);
        if (wrongTitles.length > 0) {
            console.log(`测试${i+1}: "${text}" → 错误: ${wrongTitles.join(', ')}`);
            const correction = generateCorrection(wrongTitles, yutianInfo);
            console.log(`  纠正: ${correction.split('\n')[1]}`);
        } else {
            console.log(`测试${i+1}: "${text}" → 正确`);
        }
    });
    
    // 保存检查结果
    const checkResult = {
        timestamp: new Date().toISOString(),
        yutianInfo,
        lastCheck: '称呼配置正确，需严格执行'
    };
    
    const resultPath = '/home/zhangyufeng/.openclaw/workspace/memory/yutian-name-check.json';
    fs.writeFileSync(resultPath, JSON.stringify(checkResult, null, 2));
    
    console.log(`\n✅ 检查完成，结果已保存: ${resultPath}`);
    console.log(`\n🎯 行动要求:`);
    console.log(`1. 群聊中严格使用"${yutianInfo.correctTitle}"`);
    console.log(`2. 绝对避免: ${yutianInfo.wrongTitles.join(', ')}`);
    console.log(`3. 下次群聊询问她喜欢的称呼`);
}

// 执行
if (require.main === module) {
    main();
}

module.exports = {
    extractYutianInfo,
    checkTextForWrongTitles,
    generateCorrection
};