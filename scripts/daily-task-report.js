#!/usr/bin/env node
/**
 * 每日任务汇总报告
 * 生成18:00的任务执行情况报告
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
                    const deadline = new Date(task.deadline).getTime();
                    if (deadline < now.getTime()) {
                        overdueTasks++;
                    }
                }
            }
        });
    }
    
    return {
        date: today,
        totalTasks,
        completedToday,
        failedToday,
        activeTasks,
        overdueTasks,
        completionRate: totalTasks > 0 ? ((completedToday + failedToday) / totalTasks * 100).toFixed(1) : 0
    };
}

// 获取token使用统计（需要OpenClaw API或日志）
function getTokenUsage() {
    try {
        // 尝试从session_status获取
        const statusCmd = 'openclaw session status --json 2>/dev/null || echo "{}"';
        const statusOutput = execSync(statusCmd, { encoding: 'utf-8' });
        
        let tokenInfo = { totalTokens: 0, estimatedCost: 0 };
        
        try {
            const status = JSON.parse(statusOutput);
            if (status.usage && status.usage.tokens) {
                tokenInfo.totalTokens = status.usage.tokens.total || 0;
                tokenInfo.estimatedCost = status.usage.cost || 0;
            }
        } catch (e) {
            // 如果无法解析，使用默认值
        }
        
        return tokenInfo;
    } catch (error) {
        return { totalTokens: 0, estimatedCost: 0, error: '无法获取token统计' };
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
    
    // 今日完成的任务
    if (fs.existsSync(COMPLETED_DIR)) {
        fs.readdirSync(COMPLETED_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(COMPLETED_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                
                if (task.completedAt) {
                    const completedTime = new Date(task.completedAt).getTime();
                    if (completedTime >= startOfDay) {
                        details.completed.push({
                            id: task.id,
                            description: task.description,
                            completedAt: task.completedAt,
                            completionNote: task.completionNote || '无说明'
                        });
                    }
                }
            }
        });
    }
    
    // 今日失败的任务
    if (fs.existsSync(FAILED_DIR)) {
        fs.readdirSync(FAILED_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(FAILED_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                
                if (task.failedAt) {
                    const failedTime = new Date(task.failedAt).getTime();
                    if (failedTime >= startOfDay) {
                        details.failed.push({
                            id: task.id,
                            description: task.description,
                            failedAt: task.failedAt,
                            failureReason: task.failureReason || '无说明'
                        });
                    }
                }
            }
        });
    }
    
    // 进行中任务
    if (fs.existsSync(ACTIVE_DIR)) {
        fs.readdirSync(ACTIVE_DIR).forEach(file => {
            if (file.endsWith('.json')) {
                const taskPath = path.join(ACTIVE_DIR, file);
                const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
                
                const taskDetail = {
                    id: task.id,
                    description: task.description,
                    deadline: task.deadline || '无截止时间',
                    priority: task.priority,
                    progress: task.progress || '无进度'
                };
                
                details.active.push(taskDetail);
                
                // 检查是否过期
                if (task.deadline) {
                    const deadline = new Date(task.deadline).getTime();
                    if (deadline < now.getTime()) {
                        details.overdue.push(taskDetail);
                    }
                }
            }
        });
    }
    
    return details;
}

// 生成报告
function generateReport() {
    const stats = getDailyStats();
    const tokenUsage = getTokenUsage();
    const taskDetails = getTodayTaskDetails();
    
    const now = new Date();
    const reportTime = now.toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' });
    
    let report = `📊 *每日任务汇总报告* (${stats.date})\n`;
    report += `*生成时间*: ${reportTime}\n\n`;
    
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
            report += `• ${task.description} - ${task.failureReason} (${time})\n`;
        });
        report += `\n`;
    }
    
    // 进行中任务
    if (taskDetails.active.length > 0) {
        report += `🔄 *进行中任务*\n`;
        taskDetails.active.forEach(task => {
            const priorityIcon = task.priority === 'high' ? '🔥' : task.priority === 'medium' ? '⚠️' : '📌';
            report += `${priorityIcon} ${task.description}\n`;
            report += `  进度: ${task.progress}\n`;
            report += `  截止: ${task.deadline}\n`;
        });
        report += `\n`;
    }
    
    // 过期任务
    if (taskDetails.overdue.length > 0) {
        report += `⏰ *过期任务 (需关注)*\n`;
        taskDetails.overdue.forEach(task => {
            report += `• ${task.description}\n`;
            report += `  应于: ${task.deadline}\n`;
        });
        report += `\n`;
    }
    
    // 建议
    report += `🎯 *建议*\n`;
    if (stats.overdueTasks > 0) {
        report += `• 优先处理 ${stats.overdueTasks} 个过期任务\n`;
    }
    if (stats.activeTasks > 5) {
        report += `• 任务较多 (${stats.activeTasks})，考虑调整优先级\n`;
    }
    if (stats.completionRate < 50) {
        report += `• 完成率较低 (${stats.completionRate}%)，检查任务可行性\n`;
    }
    if (stats.completionRate >= 80) {
        report += `• 完成率良好 (${stats.completionRate}%)，继续保持\n`;
    }
    
    report += `\n📅 *明日重点*\n`;
    report += `• 继续监控进行中任务\n`;
    report += `• 处理过期任务\n`;
    report += `• 优化高优先级任务执行\n`;
    
    return report;
}

// 保存报告到文件
function saveReport(report) {
    const now = new Date();
    const dateStr = now.toISOString().split('T')[0];
    const timeStr = now.toTimeString().split(' ')[0].replace(/:/g, '-');
    
    const filename = `task-report-${dateStr}-${timeStr}.txt`;
    const filepath = path.join(REPORTS_DIR, filename);
    
    fs.writeFileSync(filepath, report);
    console.log(`报告已保存: ${filepath}`);
    
    return filepath;
}

// 主函数
function main() {
    console.log('📊 生成每日任务汇总报告...');
    
    const report = generateReport();
    const savedPath = saveReport(report);
    
    console.log('\n' + report);
    console.log(`\n✅ 报告生成完成`);
    console.log(`文件: ${savedPath}`);
    
    return {
        report,
        filepath: savedPath
    };
}

// CLI接口
if (require.main === module) {
    const result = main();
    
    // 如果通过cron调用，可能需要发送到Telegram
    const args = process.argv.slice(2);
    if (args.includes('--send')) {
        console.log('📤 发送报告到Telegram...');
        // 这里可以添加发送逻辑
    }
}

module.exports = {
    generateReport,
    saveReport,
    getDailyStats,
    getTokenUsage
};