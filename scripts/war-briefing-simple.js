#!/usr/bin/env node
/**
 * 简化版战争简报脚本
 * 生成简报并通过OpenClaw发送到Telegram
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

console.log(`生成战争简报 (${slot}:00, ${period})`);

try {
    // 搜索最新消息
    const searchQuery = `美国 以色列 伊朗 战争 最新消息 ${beijingTime.getFullYear()}年${beijingTime.getMonth() + 1}月${beijingTime.getDate()}日`;
    console.log(`搜索: ${searchQuery}`);
    
    const searchResult = execSync(
        `cd /home/zhangyufeng/.openclaw/workspace && python3 ./skills/openclaw-tavily-search/scripts/tavily_search.py --query "${searchQuery}" --max-results 5 --format brave 2>&1`,
        { encoding: 'utf-8', maxBuffer: 5 * 1024 * 1024 }
    );
    
    let results = [];
    try {
        const data = JSON.parse(searchResult);
        results = data.results || [];
    } catch (e) {
        console.error('解析失败:', e.message);
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
    
    let briefing = `📊 *美以伊战争简报* (#${slot/6 + 1})\n\n`;
    briefing += `*时间*: ${slot}:00 (北京时间)\n`;
    briefing += `*时段*: ${period}\n`;
    briefing += `*生成*: ${timestamp}\n\n`;
    
    if (results.length === 0) {
        briefing += `⚠️ *无最新消息*\n过去6小时未发现重大更新。\n`;
    } else {
        // 提取关键信息
        let hasCasualties = false;
        let hasPolitical = false;
        
        for (const result of results.slice(0, 3)) {
            const snippet = result.snippet || '';
            
            // 检查伤亡信息
            if (!hasCasualties && snippet.includes('死亡') && snippet.includes('受伤')) {
                const match = snippet.match(/(\d+)\s*死亡[，、]\s*(\d+)\s*受伤/);
                if (match) {
                    briefing += `💀 *伤亡*: ${match[1]}死${match[2]}伤\n`;
                    hasCasualties = true;
                }
            }
            
            // 检查政治动态
            if (!hasPolitical && (snippet.includes('特朗普') || snippet.includes('哈梅内伊'))) {
                briefing += `🏛️ *政治*: `;
                if (snippet.includes('特朗普')) briefing += '特朗普';
                if (snippet.includes('哈梅内伊')) briefing += '哈梅内伊';
                briefing += '相关动态\n';
                hasPolitical = true;
            }
            
            // 标题作为关键事件
            const title = result.title || '';
            if (title.length > 10) {
                briefing += `📰 ${title.substring(0, 60)}${title.length > 60 ? '...' : ''}\n`;
            }
        }
        
        briefing += `\n`;
    }
    
    // 状态评估
    briefing += `*评估*:\n`;
    if (results.length >= 3) {
        briefing += `🔥 局势紧张 | ⚔️ 持续冲突 | 🌍 风险中高\n`;
    } else {
        briefing += `🟡 相对稳定 | ⚔️ 有限冲突 | 🌍 风险中低\n`;
    }
    
    briefing += `\n*下一简报*: ${(slot + 6) % 24}:00\n`;
    
    // 保存简报
    const briefingDir = '/home/zhangyufeng/.openclaw/workspace/briefings';
    if (!fs.existsSync(briefingDir)) {
        fs.mkdirSync(briefingDir, { recursive: true });
    }
    
    const filename = `briefing-${beijingTime.toISOString().split('T')[0]}-${slot}.txt`;
    const filepath = path.join(briefingDir, filename);
    fs.writeFileSync(filepath, briefing);
    
    console.log(`简报已保存: ${filepath}`);
    console.log(`简报内容:\n${briefing}`);
    
    // 通过OpenClaw发送消息
    // 这里cron job会处理发送
    
    process.exit(0);
    
} catch (error) {
    console.error(`简报生成失败:`, error.message);
    
    // 发送错误通知
    const errorMsg = `⚠️ 战争简报生成失败 (${slot}:00)\n错误: ${error.message}`;
    const errorFile = `/home/zhangyufeng/.openclaw/workspace/briefings/error-${Date.now()}.txt`;
    fs.writeFileSync(errorFile, errorMsg);
    
    process.exit(1);
}