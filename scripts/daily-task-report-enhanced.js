#!/usr/bin/env node
/**
 * 增强版每日任务汇总报告
 * 生成18:00的任务执行情况报告，并支持发送到指定群聊
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const TASKS_DIR = '/home/zhangyufeng/.openclaw/workspace/tasks';
const ACTIVE_DIR = path.join(TASKS_DIR, 'active');
const COMPLETED_DIR = path.join(TASKS_DIR, 'completed');
const FAILED_DIR = path.join(TASKS_DIR, 'failed');
const REPORTS_DIR = path.join(TASKS_DIR, 'reports');

// 确保报告目录存在
if (!fs.existsSync(REPORTS_DIR)) {
    fs.mkdirSync(REPORTS_DIR, { recursive: true });
}

// 获取当日任务统计
function getDailyStats() {
    const now = new Date();
    const today = now.toISOString().split('T')[0]; // YYYY-MM-DD
    const startOfDay = new Date(today + 'T00:00:00+08:00').getTime();
    
    let totalTasks = 0;
    let completedToday = 0;
    let failedToday = 0;
    let activeTasks = 0;
    let overdueTasks = 0;
    
    // 检查已完成任务（今日）
    if (fs.existsSync(COMPLETED_DIR)) {
        fs.readdirSync(COMPLETED_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(COMPLETED_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                totalTasks++;
                
                if (task.completedAt) {
                    const completedTime = new Date(task.completedAt).getTime();
                    if (completedTime >= startOfDay) {
                        completedToday++;
                    }
                }
            }
        });
    }
    
    // 检查失败任务（今日）
    if (fs.existsSync(FAILED_DIR)) {
        fs.readdirSync(FAILED_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(FAILED_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                totalTasks++;
                
                if (task.failedAt) {
                    const failedTime = new Date(task.failedAt).getTime();
                    if (failedTime >= startOfDay) {
                        failedToday++;
                    }
                }
            }
        });
    }
    
    // 检查进行中任务
    if (fs.existsSync(ACTIVE_DIR)) {
        fs.readdirSync(ACTIVE_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(ACTIVE_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                totalTasks++;
                activeTasks++;
                
                // 检查是否过期
                if (task.deadline) {
                    const deadlineTime = new Date(task.deadline).getTime();
                    if (deadlineTime < now.getTime()) {
                        overdueTasks++;
                    }
                }
            }
        });
    }
    
    const completionRate = totalTasks > 0 ? ((completedToday / totalTasks) * 100).toFixed(1) : '0.0';
    
    return {
        totalTasks,
        completedToday,
        failedToday,
        activeTasks,
        overdueTasks,
        completionRate
    };
}

// 获取token使用统计
function getTokenUsage() {
    try {
        // 调用token统计脚本
        const tokenStatsScript = path.join(__dirname, 'token-stats.js');
        
        if (!fs.existsSync(tokenStatsScript)) {
            return { totalTokens: 0, estimatedCost: 0, error: 'token统计脚本不存在' };
        }
        
        // 运行token统计脚本
        const output = execSync(`node "${tokenStatsScript}"`, { 
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'pipe']
        });
        
        // 从输出中提取token信息
        let totalTokens = 0;
        let estimatedCost = 0;
        
        // 解析输出中的关键信息
        const tokenMatch = output.match(/总 Token: (\d+)/);
        const costMatch = output.match(/约 (.+?) 元/);
        
        if (tokenMatch) {
            totalTokens = parseInt(tokenMatch[1].replace(/,/g, ''), 10);
        }
        
        if (costMatch) {
            estimatedCost = parseFloat(costMatch[1]);
        }
        
        return { 
            totalTokens, 
            estimatedCost,
            success: true
        };
        
    } catch (error) {
        console.error('Token统计失败:', error.message);
        return { 
            totalTokens: 0, 
            estimatedCost: 0, 
            error: 'token统计执行失败',
            success: false
        };
    }
}

// 获取今日重要任务详情
function getTodayTaskDetails() {
    const now = new Date();
    const today = now.toISOString().split('T')[0];
    const startOfDay = new Date(today + 'T00:00:00+08:00').getTime();
    
    const details = {
        completed: [],
        failed: [],
        active: [],
        overdue: []
    };
    
    // 获取已完成任务详情
    if (fs.existsSync(COMPLETED_DIR)) {
        fs.readdirSync(COMPLETED_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(COMPLETED_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                
                if (task.completedAt) {
                    const completedTime = new Date(task.completedAt).getTime();
                    if (completedTime >= startOfDay) {
                        details.completed.push({
                            description: task.description || '未命名任务',
                            completedAt: task.completedAt
                        });
                    }
                }
            }
        });
    }
    
    // 获取失败任务详情
    if (fs.existsSync(FAILED_DIR)) {
        fs.readdirSync(FAILED_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(FAILED_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                
                if (task.failedAt) {
                    const failedTime = new Date(task.failedAt).getTime();
                    if (failedTime >= startOfDay) {
                        details.failed.push({
                            description: task.description || '未命名任务',
                            failedAt: task.failedAt,
                            error: task.error || '未知错误'
                        });
                    }
                }
            }
        });
    }
    
    // 获取进行中任务详情
    if (fs.existsSync(ACTIVE_DIR)) {
        fs.readdirSync(ACTIVE_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(ACTIVE_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                
                const isOverdue = task.deadline && new Date(task.deadline).getTime() < now.getTime();
                
                const taskDetail = {
                    description: task.description || '未命名任务',
                    deadline: task.deadline,
                    progress: task.progress || '无进度信息',
                    priority: task.priority || 'normal'
                };
                
                if (isOverdue) {
                    details.overdue.push(taskDetail);
                } else {
                    details.active.push(taskDetail);
                }
            }
        });
    }
    
    return details;
}

// 生成报告
function generateReport() {
    const now = new Date();
    const beijingTime = new Date(now.getTime() + 8 * 60 * 60 * 1000);
    const timeStr = beijingTime.toLocaleString('zh-CN', { 
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
    
    const stats = getDailyStats();
    const tokenUsage = getTokenUsage();
    const taskDetails = getTodayTaskDetails();
    
    let report = `📊 *每日任务汇总报告* (${now.toISOString().split('T')[0]})\n`;
    report += `*生成时间*: ${timeStr}\n\n`;
    
    // 总体统计
    report += `📈 *总体统计*\n`;
    report += `• 总任务数: ${stats.totalTasks}\n`;
    report += `• 今日完成: ${stats.completedToday}\n`;
    report += `• 今日失败: ${stats.failedToday}\n`;
    report += `• 进行中: ${stats.activeTasks}\n`;
    report += `• 已过期: ${stats.overdueTasks}\n`;
    report += `• 完成率: ${stats.completionRate}%\n\n`;
    
    // Token使用
    report += `💸 *资源使用*\n`;
    report += `• 今日Token: ${tokenUsage.totalTokens.toLocaleString()}\n`;
    report += `• 估算成本: ¥${tokenUsage.estimatedCost.toFixed(4)}\n`;
    if (tokenUsage.error) {
        report += `• 备注: ${tokenUsage.error}\n`;
    }
    report += `\n`;
    
    // 今日完成的任务
    if (taskDetails.completed.length > 0) {
        report += `✅ *今日完成*\n`;
        taskDetails.completed.forEach(task => {
            const time = new Date(task.completedAt).toLocaleTimeString('zh-CN', { timeZone: 'Asia/Shanghai' });
            report += `• ${task.description} (${time})\n`;
        });
        report += `\n`;
    }
    
    // 今日失败的任务
    if (taskDetails.failed.length > 0) {
        report += `❌ *今日失败*\n`;
        taskDetails.failed.forEach(task => {
            const time = new Date(task.failedAt).toLocaleTimeString('zh-CN', { timeZone: 'Asia/Shanghai' });
            report += `• ${task.description} (${time})\n`;
            report += `  错误: ${task.error}\n`;
        });
        report += `\n`;
    }
    
    // 进行中任务
    if (taskDetails.active.length > 0) {
        report += `🔄 *进行中任务*\n`;
        taskDetails.active.forEach((task, index) => {
            const emoji = task.priority === 'high' ? '🔥' : task.priority === 'critical' ? '🚨' : '📝';
            report += `${emoji} ${task.description}\n`;
            report += `  进度: ${task.progress}\n`;
            if (task.deadline) {
                const deadlineStr = new Date(task.deadline).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' });
                report += `  截止: ${deadlineStr}\n`;
            }
            if (index < taskDetails.active.length - 1) report += `\n`;
        });
        report += `\n`;
    }
    
    // 过期任务
    if (taskDetails.overdue.length > 0) {
        report += `⏰ *过期任务 (需关注)*\n`;
        taskDetails.overdue.forEach(task => {
            const deadlineStr = new Date(task.deadline).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' });
            report += `• ${task.description}\n`;
            report += `  应于: ${deadlineStr}\n`;
        });
        report += `\n`;
    }
    
    // 建议
    report += `🎯 *建议*\n`;
    if (stats.overdueTasks > 0) {
        report += `• 优先处理 ${stats.overdueTasks} 个过期任务\n`;
    }
    if (parseFloat(stats.completionRate) < 50) {
        report += `• 完成率较低 (${stats.completionRate}%)，检查任务可行性\n`;
    }
    if (stats.activeTasks > 5) {
        report += `• 进行中任务较多 (${stats.activeTasks})，考虑调整优先级\n`;
    }
    if (stats.totalTasks === 0) {
        report += `• 暂无任务记录，开始创建新任务\n`;
    }
    report += `\n`;
    
    // 明日重点
    report += `📅 *明日重点*\n`;
    report += `• 继续监控进行中任务\n`;
    if (stats.overdueTasks > 0) {
        report += `• 处理 ${stats.overdueTasks} 个过期任务\n`;
    }
    report += `• 优化高优先级任务执行\n`;
    report += `• 定期检查资源使用情况\n`;
    
    return report;
}

// 保存报告到文件
function saveReport(report) {
    const now = new Date();
    const dateStr = now.toISOString().split('T')[0];
    const timeStr = now.toTimeString().split(' ')[0].replace(/:/g, '-');
    
    const filename = `task-report-enhanced-${dateStr}-${timeStr}.txt`;
    const filepath = path.join(REPORTS_DIR, filename);
    
    fs.writeFileSync(filepath, report);
    console.log(`报告已保存: ${filepath}`);
    
    return filepath;
}

// 发送报告到Telegram群聊
function sendToTelegram(report, targetChatId = '-5149902750') {
    try {
        console.log(`📤 发送报告到Telegram群聊 (ID: ${targetChatId})...`);
        
        // 使用OpenClaw message工具发送
        const sendCmd = `openclaw message send --channel telegram --target ${targetChatId} --message "${report.replace(/"/g, '\\"').replace(/\n/g, '\\n')}"`;
        
        execSync(sendCmd, { stdio: 'pipe' });
        console.log(`✅ 报告已发送到群聊`);
        
        return true;
    } catch (error) {
        console.error(`❌ 发送到Telegram失败:`, error.message);
        return false;
    }
}

// 主函数
function main() {
    console.log('📊 生成增强版每日任务汇总报告...');
    
    const report = generateReport();
    const savedPath = saveReport(report);
    
    console.log('\n' + report);
    console.log(`\n✅ 报告生成完成`);
    console.log(`文件: ${savedPath}`);
    
    // 自动发送到张府群聊
    const sendSuccess = sendToTelegram(report, '-5149902750');
    
    return {
        report,
        filepath: savedPath,
        sentToTelegram: sendSuccess
    };
}

// CLI接口
if (require.main === module) {
    const result = main();
    
    // 退出码
    if (result.sentToTelegram) {
        process.exit(0);
    } else {
        process.exit(1);
    }
}

module.exports = {
    generateReport,
    saveReport,
    sendToTelegram,
    getDailyStats,
    getTokenUsage
};