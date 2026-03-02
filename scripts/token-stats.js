#!/usr/bin/env node
/**
 * Token 使用情况统计脚本
 * 统计 @zyf_weekly_report_bot 的 token 使用情况
 */

const fs = require('fs');
const path = require('path');

async function getTokenStats() {
    console.log(`[${new Date().toISOString()}] 开始统计 token 使用情况...`);
    
    const stats = {
        weekly_report_bot: {
            sessions: {},
            totals: {
                inputTokens: 0,
                outputTokens: 0,
                cacheRead: 0,
                totalTokens: 0
            }
        },
        timestamp: new Date().toISOString()
    };
    
    try {
        // 1. 读取 weekly_report_helper 的会话文件
        const sessionsPath = '/home/zhangyufeng/.openclaw/agents/weekly_report_helper/sessions/sessions.json';
        
        if (fs.existsSync(sessionsPath)) {
            const sessionsData = JSON.parse(fs.readFileSync(sessionsPath, 'utf8'));
            
            for (const [sessionKey, sessionInfo] of Object.entries(sessionsData)) {
                if (sessionInfo.inputTokens !== undefined) {
                    stats.weekly_report_bot.sessions[sessionKey] = {
                        inputTokens: sessionInfo.inputTokens || 0,
                        outputTokens: sessionInfo.outputTokens || 0,
                        cacheRead: sessionInfo.cacheRead || 0,
                        totalTokens: sessionInfo.totalTokens || 0,
                        updatedAt: sessionInfo.updatedAt ? new Date(sessionInfo.updatedAt).toISOString() : null,
                        model: sessionInfo.model || 'unknown'
                    };
                    
                    // 累加总计
                    stats.weekly_report_bot.totals.inputTokens += sessionInfo.inputTokens || 0;
                    stats.weekly_report_bot.totals.outputTokens += sessionInfo.outputTokens || 0;
                    stats.weekly_report_bot.totals.cacheRead += sessionInfo.cacheRead || 0;
                    stats.weekly_report_bot.totals.totalTokens += sessionInfo.totalTokens || 0;
                }
            }
        }
        
        // 2. 估算费用（基于 DeepSeek 标准定价）
        // 输入: $0.14/百万 tokens
        // 输出: $0.28/百万 tokens
        // 缓存读取: $0.02/百万 tokens
        const inputCost = (stats.weekly_report_bot.totals.inputTokens / 1000000) * 0.14;
        const outputCost = (stats.weekly_report_bot.totals.outputTokens / 1000000) * 0.28;
        const cacheCost = (stats.weekly_report_bot.totals.cacheRead / 1000000) * 0.02;
        const totalCostUSD = inputCost + outputCost + cacheCost;
        const totalCostCNY = totalCostUSD * 7.2; // 假设汇率 7.2
        
        stats.weekly_report_bot.totals.estimatedCost = {
            usd: totalCostUSD,
            cny: totalCostCNY,
            inputCostUSD: inputCost,
            outputCostUSD: outputCost,
            cacheCostUSD: cacheCost
        };
        
        // 3. 生成报告
        const report = {
            timestamp: stats.timestamp,
            summary: {
                totalSessions: Object.keys(stats.weekly_report_bot.sessions).length,
                totalInputTokens: stats.weekly_report_bot.totals.inputTokens,
                totalOutputTokens: stats.weekly_report_bot.totals.outputTokens,
                totalCacheRead: stats.weekly_report_bot.totals.cacheRead,
                totalTokens: stats.weekly_report_bot.totals.totalTokens,
                estimatedCostUSD: totalCostUSD.toFixed(6),
                estimatedCostCNY: totalCostCNY.toFixed(4)
            },
            sessions: stats.weekly_report_bot.sessions,
            details: stats.weekly_report_bot.totals
        };
        
        console.log(`[${new Date().toISOString()}] Token 统计完成:`);
        console.log(`  会话数量: ${report.summary.totalSessions}`);
        console.log(`  输入 Token: ${report.summary.totalInputTokens}`);
        console.log(`  输出 Token: ${report.summary.totalOutputTokens}`);
        console.log(`  缓存读取: ${report.summary.totalCacheRead}`);
        console.log(`  总 Token: ${report.summary.totalTokens}`);
        console.log(`  估算费用: $${report.summary.estimatedCostUSD} (约 ${report.summary.estimatedCostCNY} 元)`);
        
        return report;
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 统计失败:`, error.message);
        return {
            timestamp: stats.timestamp,
            error: error.message,
            summary: {
                totalSessions: 0,
                totalInputTokens: 0,
                totalOutputTokens: 0,
                totalCacheRead: 0,
                totalTokens: 0,
                estimatedCostUSD: '0.000000',
                estimatedCostCNY: '0.0000'
            }
        };
    }
}

// 保存统计报告到文件
async function saveReport(report) {
    const reportsDir = '/home/zhangyufeng/.openclaw/workspace/token-reports';
    
    if (!fs.existsSync(reportsDir)) {
        fs.mkdirSync(reportsDir, { recursive: true });
    }
    
    const dateStr = new Date().toISOString().split('T')[0];
    const reportFile = path.join(reportsDir, `token-stats-${dateStr}.json`);
    
    // 读取现有报告（如果存在）
    let dailyReports = [];
    if (fs.existsSync(reportFile)) {
        try {
            dailyReports = JSON.parse(fs.readFileSync(reportFile, 'utf8'));
            if (!Array.isArray(dailyReports)) {
                dailyReports = [dailyReports];
            }
        } catch (e) {
            dailyReports = [];
        }
    }
    
    // 添加新报告
    dailyReports.push(report);
    
    // 只保留最近30天的报告
    if (dailyReports.length > 30) {
        dailyReports = dailyReports.slice(-30);
    }
    
    // 保存到文件
    fs.writeFileSync(reportFile, JSON.stringify(dailyReports, null, 2));
    console.log(`[${new Date().toISOString()}] 报告已保存: ${reportFile}`);
    
    return reportFile;
}

// 生成文本格式的汇总报告
function generateTextReport(report) {
    const date = new Date(report.timestamp);
    const dateStr = date.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' });
    
    let text = `📊 @zyf_weekly_report_bot Token 使用情况统计\n`;
    text += `📅 统计时间: ${dateStr}\n\n`;
    
    text += `📈 汇总统计:\n`;
    text += `├─ 会话数量: ${report.summary.totalSessions}\n`;
    text += `├─ 输入 Token: ${report.summary.totalInputTokens.toLocaleString()}\n`;
    text += `├─ 输出 Token: ${report.summary.totalOutputTokens.toLocaleString()}\n`;
    text += `├─ 缓存读取: ${report.summary.totalCacheRead.toLocaleString()}\n`;
    text += `├─ 总 Token: ${report.summary.totalTokens.toLocaleString()}\n`;
    text += `├─ 估算费用: $${report.summary.estimatedCostUSD}\n`;
    text += `└─ 约 ${report.summary.estimatedCostCNY} 元\n\n`;
    
    // 按会话详细统计
    if (Object.keys(report.sessions).length > 0) {
        text += `📋 会话详情:\n`;
        let index = 1;
        for (const [sessionKey, session] of Object.entries(report.sessions)) {
            const shortKey = sessionKey.length > 40 ? sessionKey.substring(0, 40) + '...' : sessionKey;
            const updatedAt = session.updatedAt ? new Date(session.updatedAt).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) : '未知';
            
            text += `${index}. ${shortKey}\n`;
            text += `   ├─ 输入: ${session.inputTokens.toLocaleString()}\n`;
            text += `   ├─ 输出: ${session.outputTokens.toLocaleString()}\n`;
            text += `   ├─ 缓存: ${session.cacheRead.toLocaleString()}\n`;
            text += `   ├─ 总计: ${session.totalTokens.toLocaleString()}\n`;
            text += `   ├─ 模型: ${session.model}\n`;
            text += `   └─ 更新时间: ${updatedAt}\n`;
            index++;
        }
    }
    
    text += `\n💡 说明:\n`;
    text += `- 基于 DeepSeek 标准定价估算\n`;
    text += `- 输入: $0.14/百万 tokens\n`;
    text += `- 输出: $0.28/百万 tokens\n`;
    text += `- 缓存读取: $0.02/百万 tokens\n`;
    text += `- 汇率: 1 USD = 7.2 CNY\n`;
    
    return text;
}

// 主函数
async function main() {
    try {
        const report = await getTokenStats();
        const reportFile = await saveReport(report);
        const textReport = generateTextReport(report);
        
        // 保存文本报告
        const textReportFile = reportFile.replace('.json', '.txt');
        fs.writeFileSync(textReportFile, textReport);
        
        console.log(`[${new Date().toISOString()}] 文本报告已保存: ${textReportFile}`);
        
        // 输出到控制台
        console.log('\n' + textReport);
        
        return {
            success: true,
            reportFile: reportFile,
            textReportFile: textReportFile,
            summary: report.summary
        };
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 执行失败:`, error);
        return {
            success: false,
            error: error.message
        };
    }
}

// 如果是直接运行
if (require.main === module) {
    main().then(result => {
        if (result.success) {
            process.exit(0);
        } else {
            process.exit(1);
        }
    });
}

module.exports = {
    getTokenStats,
    saveReport,
    generateTextReport,
    main
};