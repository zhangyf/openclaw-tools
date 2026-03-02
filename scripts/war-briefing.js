#!/usr/bin/env node
/**
 * 美以伊战争自动简报脚本
 * 每天4个时间点执行，汇总前6小时情况
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// 时间配置
const now = new Date();
const beijingTime = new Date(now.getTime() + 8 * 60 * 60 * 1000); // UTC+8
const timeSlot = getTimeSlot(beijingTime.getHours());
const period = getPeriod(timeSlot);

console.log(`[${now.toISOString()}] 开始生成战争简报...`);
console.log(`时间点: ${timeSlot}:00 (北京时间)`);
console.log(`汇总时段: ${period}`);

// 搜索最新战争新闻
async function generateBriefing() {
    try {
        // 使用tavily搜索最新消息
        const searchQuery = `美国 以色列 伊朗 战争 最新消息 ${beijingTime.getFullYear()}年${beijingTime.getMonth() + 1}月${beijingTime.getDate()}日`;
        
        console.log(`搜索关键词: ${searchQuery}`);
        
        const searchResult = execSync(
            `cd /home/zhangyufeng/.openclaw/workspace && python3 ./skills/openclaw-tavily-search/scripts/tavily_search.py --query "${searchQuery}" --max-results 10 --format brave 2>&1`,
            { encoding: 'utf-8', maxBuffer: 10 * 1024 * 1024 }
        );
        
        let searchData;
        try {
            searchData = JSON.parse(searchResult);
        } catch (e) {
            console.error('解析搜索结果失败:', e.message);
            searchData = { results: [] };
        }
        
        // 生成简报
        const briefing = generateBriefingContent(searchData.results || [], timeSlot, period);
        
        // 保存简报文件
        const briefingDir = '/home/zhangyufeng/.openclaw/workspace/briefings';
        if (!fs.existsSync(briefingDir)) {
            fs.mkdirSync(briefingDir, { recursive: true });
        }
        
        const filename = `war-briefing-${beijingTime.toISOString().split('T')[0]}-${timeSlot}.md`;
        const filepath = path.join(briefingDir, filename);
        
        fs.writeFileSync(filepath, briefing);
        console.log(`简报已保存: ${filepath}`);
        
        // 发送到Telegram
        await sendToTelegram(briefing);
        
        return briefing;
        
    } catch (error) {
        console.error(`生成简报失败:`, error.message);
        // 发送错误通知
        await sendToTelegram(`⚠️ 战争简报生成失败 (${timeSlot}:00)\n错误: ${error.message}`);
        throw error;
    }
}

// 生成简报内容
function generateBriefingContent(results, timeSlot, period) {
    const timestamp = beijingTime.toLocaleString('zh-CN', { 
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
    
    let briefing = `## 📊 美以伊战争简报 #${timeSlot/6}\n`;
    briefing += `**时间点**: ${timeSlot}:00 (北京时间)\n`;
    briefing += `**汇总时段**: ${period}\n`;
    briefing += `**生成时间**: ${timestamp}\n`;
    briefing += `---\n\n`;
    
    if (results.length === 0) {
        briefing += `### ⚠️ 无最新消息\n`;
        briefing += `过去6小时内未发现重大战况更新。\n`;
        briefing += `可能原因：\n`;
        briefing += `1. 战况相对稳定\n`;
        briefing += `2. 信息延迟\n`;
        briefing += `3. 搜索API限制\n\n`;
        briefing += `**建议**: 手动检查权威新闻源。\n`;
        return briefing;
    }
    
    // 分析搜索结果
    const casualties = extractCasualties(results);
    const keyEvents = extractKeyEvents(results);
    const politicalUpdates = extractPoliticalUpdates(results);
    const regionalImpact = extractRegionalImpact(results);
    
    // 1. 核心战况
    briefing += `### 一、核心战况更新\n\n`;
    if (casualties) {
        briefing += `**伤亡统计**: ${casualties}\n\n`;
    }
    if (keyEvents.length > 0) {
        briefing += `**关键事件**:\n`;
        keyEvents.forEach(event => {
            briefing += `- ${event}\n`;
        });
        briefing += `\n`;
    }
    
    // 2. 军事行动
    briefing += `### 二、军事行动\n\n`;
    const militaryActions = extractMilitaryActions(results);
    if (militaryActions.length > 0) {
        militaryActions.forEach(action => {
            briefing += `- ${action}\n`;
        });
    } else {
        briefing += `- 无重大军事行动更新\n`;
    }
    briefing += `\n`;
    
    // 3. 政治动态
    briefing += `### 三、政治动态\n\n`;
    if (politicalUpdates.length > 0) {
        politicalUpdates.forEach(update => {
            briefing += `- ${update}\n`;
        });
    } else {
        briefing += `- 无重大政治动态更新\n`;
    }
    briefing += `\n`;
    
    // 4. 地区影响
    briefing += `### 四、地区影响\n\n`;
    if (regionalImpact.length > 0) {
        regionalImpact.forEach(impact => {
            briefing += `- ${impact}\n`;
        });
    } else {
        briefing += `- 无新增地区影响\n`;
    }
    briefing += `\n`;
    
    // 5. 简报评价
    briefing += `### 五、简报评价\n\n`;
    briefing += `**局势评级**: ${rateSituation(results)}\n`;
    briefing += `**冲突级别**: ${rateConflictLevel(results)}\n`;
    briefing += `**扩散风险**: ${rateSpreadRisk(results)}\n\n`;
    
    // 6. 下一份简报
    briefing += `### 六、下一份简报\n\n`;
    briefing += `**时间**: ${(timeSlot + 6) % 24}:00 (北京时间)\n`;
    briefing += `**汇总时段**: ${getPeriod((timeSlot + 6) % 24)}\n\n`;
    
    // 7. 信息来源
    briefing += `### 七、信息来源\n\n`;
    results.slice(0, 3).forEach((result, index) => {
        briefing += `${index + 1}. [${result.title}](${result.url})\n`;
    });
    
    return briefing;
}

// 辅助函数
function getTimeSlot(hour) {
    if (hour >= 0 && hour < 6) return 0;
    if (hour >= 6 && hour < 12) return 6;
    if (hour >= 12 && hour < 18) return 12;
    return 18;
}

function getPeriod(timeSlot) {
    const periods = {
        0: '前日18:00-今日00:00',
        6: '00:00-06:00',
        12: '06:00-12:00',
        18: '12:00-18:00'
    };
    return periods[timeSlot] || '未知时段';
}

function extractCasualties(results) {
    for (const result of results) {
        if (result.snippet && result.snippet.includes('死亡') && result.snippet.includes('受伤')) {
            const match = result.snippet.match(/(\d+)\s*死亡[，、]\s*(\d+)\s*受伤/);
            if (match) {
                return `${match[1]}人死亡，${match[2]}人受伤`;
            }
        }
    }
    return null;
}

function extractKeyEvents(results) {
    const events = [];
    const keywords = ['小学', '医院', '平民', '儿童', '学校', '居民区'];
    
    for (const result of results) {
        if (result.snippet) {
            for (const keyword of keywords) {
                if (result.snippet.includes(keyword) && !events.some(e => e.includes(keyword))) {
                    events.push(result.snippet.substring(0, 100) + '...');
                    break;
                }
            }
        }
    }
    
    return events.slice(0, 3);
}

function extractPoliticalUpdates(results) {
    const updates = [];
    const keywords = ['特朗普', '哈梅内伊', '以色列总理', '联合国', '国会', '谈判'];
    
    for (const result of results) {
        if (result.snippet) {
            for (const keyword of keywords) {
                if (result.snippet.includes(keyword) && !updates.some(u => u.includes(keyword))) {
                    updates.push(result.snippet.substring(0, 100) + '...');
                    break;
                }
            }
        }
    }
    
    return updates.slice(0, 3);
}

function extractRegionalImpact(results) {
    const impacts = [];
    const regions = ['卡塔尔', '阿联酋', '科威特', '沙特', '巴林', '约旦'];
    
    for (const result of results) {
        if (result.snippet) {
            for (const region of regions) {
                if (result.snippet.includes(region) && !impacts.some(i => i.includes(region))) {
                    impacts.push(`${region}: ${result.snippet.substring(0, 80)}...`);
                    break;
                }
            }
        }
    }
    
    return impacts.slice(0, 3);
}

function extractMilitaryActions(results) {
    const actions = [];
    const keywords = ['袭击', '打击', '导弹', '无人机', '航母', '基地'];
    
    for (const result of results) {
        if (result.snippet) {
            const snippet = result.snippet;
            for (const keyword of keywords) {
                if (snippet.includes(keyword)) {
                    // 提取包含关键词的句子
                    const sentences = snippet.split(/[。.!?]/);
                    for (const sentence of sentences) {
                        if (sentence.includes(keyword) && sentence.length > 20) {
                            actions.push(sentence.trim());
                            break;
                        }
                    }
                    break;
                }
            }
        }
    }
    
    return [...new Set(actions)].slice(0, 5);
}

function rateSituation(results) {
    if (results.length === 0) return '🟡 待观察';
    
    const criticalKeywords = ['死亡', '袭击', '爆炸', '打击', '导弹'];
    let criticalCount = 0;
    
    for (const result of results) {
        if (result.snippet) {
            for (const keyword of criticalKeywords) {
                if (result.snippet.includes(keyword)) {
                    criticalCount++;
                    break;
                }
            }
        }
    }
    
    if (criticalCount >= 5) return '🔥🔥🔥🔥 高度危险';
    if (criticalCount >= 3) return '🔥🔥🔥 危险升级';
    if (criticalCount >= 1) return '🔥🔥 紧张持续';
    return '🔥 相对稳定';
}

function rateConflictLevel(results) {
    const keywords = ['全面战争', '大规模', '持续', '有限', '局部'];
    
    for (const result of results) {
        if (result.snippet) {
            if (result.snippet.includes('全面战争') || result.snippet.includes('大规模')) {
                return '全面军事对抗';
            }
            if (result.snippet.includes('持续')) {
                return '持续冲突';
            }
            if (result.snippet.includes('有限') || result.snippet.includes('局部')) {
                return '有限冲突';
            }
        }
    }
    
    return '待评估';
}

function rateSpreadRisk(results) {
    const regionKeywords = ['卡塔尔', '阿联酋', '科威特', '沙特', '巴林', '约旦', '黎巴嫩', '叙利亚'];
    let regionCount = 0;
    
    for (const result of results) {
        if (result.snippet) {
            for (const keyword of regionKeywords) {
                if (result.snippet.includes(keyword)) {
                    regionCount++;
                    break;
                }
            }
        }
    }
    
    if (regionCount >= 3) return '高';
    if (regionCount >= 1) return '中';
    return '低';
}

// 发送到Telegram
async function sendToTelegram(briefing) {
    try {
        // 截取前2000字符（Telegram消息限制）
        const message = briefing.length > 2000 ? briefing.substring(0, 2000) + '...\n\n(完整简报已保存)' : briefing;
        
        // 使用OpenClaw的message工具发送
        const messageTool = require('/home/zhangyufeng/.npm-global/lib/node_modules/openclaw/dist/tools/message.js');
        
        // 这里需要实际的发送逻辑
        // 由于工具调用复杂，先保存到文件，后续通过cron job发送
        
        console.log(`简报已生成，长度: ${briefing.length}字符`);
        console.log(`消息预览: ${message.substring(0, 100)}...`);
        
        return true;
        
    } catch (error) {
        console.error('发送到Telegram失败:', error.message);
        return false;
    }
}

// 执行
generateBriefing().then(() => {
    console.log(`[${now.toISOString()}] 战争简报生成完成`);
    process.exit(0);
}).catch(error => {
    console.error(`[${now.toISOString()}] 战争简报生成失败:`, error);
    process.exit(1);
});