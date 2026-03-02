#!/usr/bin/env node
/**
 * 战争简报脚本（含财经分析）
 * 加入石油、股市、经济影响分析
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// 获取当前时间
const now = new Date();
const beijingTime = new Date(now.getTime() + 8 * 60 * 60 * 1000);
const hour = beijingTime.getHours();

// 确定时间点和时段
function getTimeSlot(h) {
    if (h >= 0 && h < 6) return { slot: 0, period: '前日18:00-今日00:00' };
    if (h >= 6 && h < 12) return { slot: 6, period: '00:00-06:00' };
    if (h >= 12 && h < 18) return { slot: 12, period: '06:00-12:00' };
    return { slot: 18, period: '12:00-18:00' };
}

const { slot, period } = getTimeSlot(hour);

console.log(`生成战争财经简报 (${slot}:00, ${period})`);

try {
    // 搜索战争新闻 - 更精准的关键词
    const warQuery = `伊朗官方确认哈梅内伊 死亡 最新消息 ${beijingTime.getFullYear()}年${beijingTime.getMonth() + 1}月${beijingTime.getDate()}日 霍尔木兹海峡 封锁`;
    console.log(`搜索战争新闻: ${warQuery}`);
    
    const warResult = execSync(
        `cd /home/zhangyufeng/.openclaw/workspace && python3 ./skills/openclaw-tavily-search/scripts/tavily_search.py --query "${warQuery}" --max-results 5 --format brave 2>&1`,
        { encoding: 'utf-8', maxBuffer: 5 * 1024 * 1024 }
    );
    
    let warResults = [];
    try {
        const data = JSON.parse(warResult);
        warResults = data.results || [];
    } catch (e) {
        console.error('解析战争新闻失败:', e.message);
    }
    
    // 搜索财经新闻
    const financeQuery = `石油价格 暴涨 霍尔木兹海峡 封锁 股市 影响 ${beijingTime.getFullYear()}年${beijingTime.getMonth() + 1}月${beijingTime.getDate()}日`;
    console.log(`搜索财经新闻: ${financeQuery}`);
    
    const financeResult = execSync(
        `cd /home/zhangyufeng/.openclaw/workspace && python3 ./skills/openclaw-tavily-search/scripts/tavily_search.py --query "${financeQuery}" --max-results 5 --format brave 2>&1`,
        { encoding: 'utf-8', maxBuffer: 5 * 1024 * 1024 }
    );
    
    let financeResults = [];
    try {
        const data = JSON.parse(financeResult);
        financeResults = data.results || [];
    } catch (e) {
        console.error('解析财经新闻失败:', e.message);
    }
    
    // 生成简报
    const timestamp = beijingTime.toLocaleString('zh-CN', {
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
    
    let briefing = `📊 *美以伊战争财经简报* (#${slot/6 + 1})\n\n`;
    briefing += `*时间*: ${slot}:00 (北京时间)\n`;
    briefing += `*时段*: ${period}\n`;
    briefing += `*生成*: ${timestamp}\n\n`;
    
    // === 战争部分 ===
    briefing += `⚔️ *战争动态*\n`;
    
    if (warResults.length === 0) {
        briefing += `暂无重大战况更新\n`;
    } else {
        // 哈梅内伊状态 - 激进模式：有消息就报，你去求证
        let hamaneyStatus = '暂无最新消息';
        let hamaneyReport = '';
        
        for (const result of warResults) {
            const snippet = result.snippet || '';
            const title = result.title || '';
            const fullText = title + ' ' + snippet;
            
            if (fullText.includes('哈梅内伊')) {
                // 激进：只要提到哈梅内伊和死亡相关词，就报道
                const deathKeywords = ['身亡', '死亡', '遇难', '逝世', '去世', '被杀'];
                const confirmKeywords = ['确认', '证实', '宣布', '承认'];
                const denyKeywords = ['否认', '安全', '活着', '假消息'];
                
                let hasDeath = deathKeywords.some(k => fullText.includes(k));
                let hasConfirm = confirmKeywords.some(k => fullText.includes(k));
                let hasDeny = denyKeywords.some(k => fullText.includes(k));
                
                if (hasDeath) {
                    if (hasConfirm) {
                        hamaneyStatus = '✅ 伊朗官方确认身亡';
                        hamaneyReport = title.substring(0, 60);
                        break;
                    } else if (hasDeny) {
                        hamaneyStatus = '❌ 伊朗否认身亡（据称安全）';
                        hamaneyReport = title.substring(0, 60);
                        break;
                    } else {
                        // 激进：只有死亡报道，没有确认也没有否认，也报道
                        hamaneyStatus = '⚠️ 据报身亡（待你求证）';
                        hamaneyReport = title.substring(0, 60);
                        break;
                    }
                }
            }
        }
        
        briefing += `• 哈梅内伊: ${hamaneyStatus}\n`;
        if (hamaneyReport) {
            briefing += `  来源: ${hamaneyReport}...\n`;
        }
        
        // 霍尔木兹海峡 - 激进模式
        let hormuzStatus = '状态正常';
        let hormuzReport = '';
        for (const result of warResults) {
            const snippet = result.snippet || '';
            const title = result.title || '';
            if (snippet.includes('霍尔木兹') || snippet.includes('海峡') || title.includes('霍尔木兹')) {
                if (snippet.includes('封锁') || snippet.includes('关闭') || title.includes('封锁')) {
                    hormuzStatus = '🚫 据报已封锁';
                    hormuzReport = title.substring(0, 60);
                    break;
                } else if (snippet.includes('开放') || snippet.includes('通行')) {
                    hormuzStatus = '✅ 恢复通行';
                    break;
                }
            }
        }
        briefing += `• 霍尔木兹海峡: ${hormuzStatus}\n`;
        if (hormuzReport) {
            briefing += `  来源: ${hormuzReport}...\n`;
        }
        
        // 关键事件
        let keyEvent = '';
        for (const result of warResults.slice(0, 2)) {
            const title = result.title || '';
            if (title.length > 10) {
                keyEvent = title.substring(0, 50) + (title.length > 50 ? '...' : '');
                break;
            }
        }
        if (keyEvent) {
            briefing += `• ${keyEvent}\n`;
        }
    }
    
    briefing += `\n`;
    
    // === 财经部分 ===
    briefing += `💰 *财经影响*\n`;
    
    if (financeResults.length === 0) {
        briefing += `暂无财经数据更新\n`;
    } else {
        // 石油价格 - 激进模式
        let oilPrice = '📈 预计暴涨';
        let oilReport = '';
        for (const result of financeResults) {
            const snippet = result.snippet || '';
            const title = result.title || '';
            const fullText = title + ' ' + snippet;
            
            if (fullText.includes('石油') || fullText.includes('原油') || fullText.includes('油价')) {
                // 激进：只要提到涨/跌就报道
                if (fullText.includes('暴涨') || fullText.includes('飙升') || fullText.includes('大涨')) {
                    oilPrice = '🚀 据报暴涨';
                    oilReport = title.substring(0, 60);
                    break;
                } else if (fullText.includes('下跌') || fullText.includes('回落') || fullText.includes('跌')) {
                    oilPrice = '📉 据报下跌';
                    oilReport = title.substring(0, 60);
                    break;
                } else if (fullText.includes('涨')) {
                    oilPrice = '📈 据报上涨';
                    oilReport = title.substring(0, 60);
                    break;
                }
            }
        }
        briefing += `• 石油价格: ${oilPrice}\n`;
        if (oilReport) {
            briefing += `  来源: ${oilReport}...\n`;
        }
        // 市场预测
        briefing += `• 中国股市: 周一预计下跌\n`;
        briefing += `• 港股: 受冲击更大\n`;
        briefing += `• 避险资产: 黄金上涨\n`;
        
        // 关键影响
        let financeImpact = '';
        for (const result of financeResults.slice(0, 2)) {
            const title = result.title || '';
            if (title.includes('股市') || title.includes('经济') || title.includes('影响')) {
                financeImpact = title.substring(0, 60) + (title.length > 60 ? '...' : '');
                break;
            }
        }
        if (financeImpact) {
            briefing += `• ${financeImpact}\n`;
        }
    }
    
    briefing += `\n`;
    
    // === 投资建议 ===
    briefing += `🎯 *投资策略*\n`;
    briefing += `1. 减仓航空、航运股\n`;
    briefing += `2. 关注石油、黄金板块\n`;
    briefing += `3. 军工股短期机会\n`;
    briefing += `4. 新能源替代逻辑\n`;
    briefing += `5. 控制仓位，谨慎抄底\n`;
    
    briefing += `\n`;
    
    // === 评估 ===
    briefing += `📈 *市场评估*\n`;
    
    const hasCriticalNews = warResults.length > 0 && (
        warResults.some(r => (r.snippet || '').includes('封锁')) ||
        warResults.some(r => (r.snippet || '').includes('身亡'))
    );
    
    if (hasCriticalNews) {
        briefing += `🔥 高度危险 | 📉 股市看跌 | 🛢️ 石油看涨\n`;
        briefing += `⚠️ 建议: 大幅减仓，等待局势明朗\n`;
    } else {
        briefing += `🟡 中等风险 | 📊 震荡为主 | 🛢️ 石油承压\n`;
        briefing += `ℹ️ 建议: 谨慎观望，控制仓位\n`;
    }
    
    briefing += `\n`;
    
    // === 下一简报 ===
    briefing += `⏰ *下一简报*: ${(slot + 6) % 24}:00\n`;
    
    // 保存简报
    const briefingDir = '/home/zhangyufeng/.openclaw/workspace/briefings';
    if (!fs.existsSync(briefingDir)) {
        fs.mkdirSync(briefingDir, { recursive: true });
    }
    
    const filename = `briefing-finance-${beijingTime.toISOString().split('T')[0]}-${slot}.txt`;
    const filepath = path.join(briefingDir, filename);
    fs.writeFileSync(filepath, briefing);
    
    console.log(`财经简报已保存: ${filepath}`);
    console.log(`简报内容:\n${briefing}`);
    
    process.exit(0);
    
} catch (error) {
    console.error(`财经简报生成失败:`, error.message);
    
    const errorMsg = `⚠️ 财经简报生成失败 (${slot}:00)\n错误: ${error.message}`;
    const errorFile = `/home/zhangyufeng/.openclaw/workspace/briefings/error-${Date.now()}.txt`;
    fs.writeFileSync(errorFile, errorMsg);
    
    process.exit(1);
}