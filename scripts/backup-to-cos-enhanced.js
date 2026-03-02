#!/usr/bin/env node
/**
 * OpenClaw 增强版备份脚本
 * 备份主工作空间 + weekly_report_helper 所有配置 + Telegram 配置
 * 包含 @zyf_weekly_report_bot token 使用情况统计
 */

const COS = require('cos-nodejs-sdk-v5');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// 自动加载.env文件（如果环境变量未设置）
function loadEnv() {
    if (process.env.TENCENT_COS_SECRET_ID && process.env.TENCENT_COS_SECRET_KEY) {
        return; // 环境变量已存在，无需加载
    }
    
    const envPath = path.join(process.env.HOME, '.openclaw/workspace/.env');
    if (fs.existsSync(envPath)) {
        const envContent = fs.readFileSync(envPath, 'utf8');
        envContent.split('\n').forEach(line => {
            const match = line.match(/^([^#=]+)=(.*)$/);
            if (match) {
                const key = match[1].trim();
                const value = match[2].trim();
                if (!process.env[key]) {
                    process.env[key] = value;
                }
            }
        });
        console.log(`[${new Date().toISOString()}] 已从.env文件加载环境变量`);
    }
}

// 加载环境变量
loadEnv();

// COS配置 - 从环境变量读取密钥
const config = {
    Bucket: 'openclaw-bakup-1251036673',
    Region: 'ap-singapore',
    SecretId: process.env.TENCENT_COS_SECRET_ID,
    SecretKey: process.env.TENCENT_COS_SECRET_KEY
};

if (!config.SecretId || !config.SecretKey) {
    console.error(`[${new Date().toISOString()}] 错误: 请设置环境变量 TENCENT_COS_SECRET_ID 和 TENCENT_COS_SECRET_KEY`);
    console.error(`[${new Date().toISOString()}] 提示: 检查 ~/.openclaw/workspace/.env 文件是否存在且包含正确的密钥`);
    process.exit(1);
}

const cos = new COS({
    SecretId: config.SecretId,
    SecretKey: config.SecretKey
});

const WORKSPACE_DIR = '/home/zhangyufeng/.openclaw/workspace';
const BACKUP_DIR = '/tmp/openclaw-backup';

// 运行 token 统计
async function runTokenStats() {
    try {
        console.log(`[${new Date().toISOString()}] 开始统计 token 使用情况...`);
        
        // 使用 node 运行 token 统计脚本
        const tokenStatsScript = path.join(WORKSPACE_DIR, 'scripts/token-stats.js');
        if (fs.existsSync(tokenStatsScript)) {
            const result = execSync(`node "${tokenStatsScript}"`, { encoding: 'utf8' });
            console.log(`[${new Date().toISOString()}] Token 统计完成`);
            
            // 解析输出中的汇总信息
            const summaryMatch = result.match(/估算费用: \$(.+?) \(约 (.+?) 元\)/);
            if (summaryMatch) {
                return {
                    success: true,
                    usdCost: summaryMatch[1],
                    cnyCost: summaryMatch[2]
                };
            }
        } else {
            console.log(`[${new Date().toISOString()}] Token 统计脚本不存在: ${tokenStatsScript}`);
        }
    } catch (error) {
        console.error(`[${new Date().toISOString()}] Token 统计失败:`, error.message);
    }
    return { success: false };
}

// 备份核心函数
async function backup() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const dateStr = new Date().toISOString().split('T')[0];
    const timeStr = new Date().toLocaleTimeString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false });
    
    console.log(`[${new Date().toISOString()}] 开始增强版备份...`);
    
    // 运行 token 统计
    const tokenStats = await runTokenStats();
    
    // 创建临时备份目录
    if (!fs.existsSync(BACKUP_DIR)) {
        fs.mkdirSync(BACKUP_DIR, { recursive: true });
    }
    
    const tarFileName = `openclaw-enhanced-backup-${dateStr}-${timestamp}.tar.gz`;
    const tarPath = path.join(BACKUP_DIR, tarFileName);
    
    try {
        // 创建备份清单
        const backupManifest = {
            timestamp: new Date().toISOString(),
            date: dateStr,
            time: timeStr,
            components: {
                main_workspace: WORKSPACE_DIR,
                weekly_report_helper: {
                    workspace: '/home/zhangyufeng/.openclaw/workspace-weekly_report_helper',
                    agents: '/home/zhangyufeng/.openclaw/agents/weekly_report_helper',
                    memory: '/home/zhangyufeng/.openclaw/memory/weekly_report_helper.sqlite'
                },
                telegram_config: '/home/zhangyufeng/.openclaw/telegram'
            },
            files: []
        };
        
        // 创建临时目录
        const tempBackupDir = path.join(BACKUP_DIR, 'temp-enhanced-backup');
        if (fs.existsSync(tempBackupDir)) {
            fs.rmSync(tempBackupDir, { recursive: true, force: true });
        }
        fs.mkdirSync(tempBackupDir, { recursive: true });
        
        console.log(`[${new Date().toISOString()}] 收集备份文件...`);
        
        // 1. 备份主工作空间
        const mainBackupDir = path.join(tempBackupDir, 'main-workspace');
        fs.mkdirSync(mainBackupDir, { recursive: true });
        
        console.log(`[${new Date().toISOString()}] 备份主工作空间...`);
        execSync(
            `tar -czf "${path.join(mainBackupDir, 'main-workspace.tar.gz')}" -C "${WORKSPACE_DIR}" --exclude='node_modules' --exclude='.git' --exclude='briefings' --exclude='token-reports' --exclude='backup-reports' --exclude='tasks' .`,
            { stdio: 'inherit' }
        );
        
        const mainBackupSize = fs.statSync(path.join(mainBackupDir, 'main-workspace.tar.gz')).size;
        backupManifest.files.push({
            name: 'main-workspace.tar.gz',
            path: 'main-workspace/main-workspace.tar.gz',
            size: mainBackupSize
        });
        
        // 2. 备份 weekly_report_helper 工作空间
        const weeklyWorkspaceDir = '/home/zhangyufeng/.openclaw/workspace-weekly_report_helper';
        if (fs.existsSync(weeklyWorkspaceDir)) {
            const weeklyBackupDir = path.join(tempBackupDir, 'weekly-report-helper');
            fs.mkdirSync(weeklyBackupDir, { recursive: true });
            
            console.log(`[${new Date().toISOString()}] 备份 weekly_report_helper 工作空间...`);
            
            // 备份核心配置文件
            const coreFiles = [
                'AGENTS.md', 'SOUL.md', 'TOOLS.md', 'IDENTITY.md', 
                'USER.md', 'MEMORY.md', 'HEARTBEAT.md', 'BOOTSTRAP.md'
            ];
            
            for (const file of coreFiles) {
                const sourcePath = path.join(weeklyWorkspaceDir, file);
                if (fs.existsSync(sourcePath)) {
                    const destPath = path.join(weeklyBackupDir, file);
                    fs.copyFileSync(sourcePath, destPath);
                    
                    backupManifest.files.push({
                        name: file,
                        path: `weekly-report-helper/${file}`,
                        size: fs.statSync(destPath).size
                    });
                }
            }
            
            // 备份 memory 目录
            const weeklyMemoryDir = path.join(weeklyWorkspaceDir, 'memory');
            if (fs.existsSync(weeklyMemoryDir)) {
                const memoryBackupPath = path.join(weeklyBackupDir, 'memory.tar.gz');
                execSync(
                    `tar -czf "${memoryBackupPath}" -C "${weeklyMemoryDir}" .`,
                    { stdio: 'inherit' }
                );
                
                backupManifest.files.push({
                    name: 'memory.tar.gz',
                    path: 'weekly-report-helper/memory.tar.gz',
                    size: fs.statSync(memoryBackupPath).size
                });
            }
            
            // 备份 weekly_summaries 目录
            const summariesDir = path.join(weeklyWorkspaceDir, 'weekly_summaries');
            if (fs.existsSync(summariesDir)) {
                const summariesBackupPath = path.join(weeklyBackupDir, 'weekly_summaries.tar.gz');
                execSync(
                    `tar -czf "${summariesBackupPath}" -C "${summariesDir}" .`,
                    { stdio: 'inherit' }
                );
                
                backupManifest.files.push({
                    name: 'weekly_summaries.tar.gz',
                    path: 'weekly-report-helper/weekly_summaries.tar.gz',
                    size: fs.statSync(summariesBackupPath).size
                });
            }
        }
        
        // 3. 备份 weekly_report_helper 代理配置
        const agentsDir = '/home/zhangyufeng/.openclaw/agents/weekly_report_helper';
        if (fs.existsSync(agentsDir)) {
            const agentsBackupDir = path.join(tempBackupDir, 'weekly-report-agent');
            fs.mkdirSync(agentsBackupDir, { recursive: true });
            
            console.log(`[${new Date().toISOString()}] 备份 weekly_report_helper 代理配置...`);
            
            // 备份 sessions 目录
            const sessionsDir = path.join(agentsDir, 'sessions');
            if (fs.existsSync(sessionsDir)) {
                const sessionsBackupPath = path.join(agentsBackupDir, 'sessions.tar.gz');
                execSync(
                    `tar -czf "${sessionsBackupPath}" -C "${sessionsDir}" .`,
                    { stdio: 'inherit' }
                );
                
                backupManifest.files.push({
                    name: 'sessions.tar.gz',
                    path: 'weekly-report-agent/sessions.tar.gz',
                    size: fs.statSync(sessionsBackupPath).size
                });
            }
            
            // 备份 agent 目录
            const agentDir = path.join(agentsDir, 'agent');
            if (fs.existsSync(agentDir)) {
                const agentBackupPath = path.join(agentsBackupDir, 'agent.tar.gz');
                execSync(
                    `tar -czf "${agentBackupPath}" -C "${agentDir}" .`,
                    { stdio: 'inherit' }
                );
                
                backupManifest.files.push({
                    name: 'agent.tar.gz',
                    path: 'weekly-report-agent/agent.tar.gz',
                    size: fs.statSync(agentBackupPath).size
                });
            }
        }
        
        // 4. 备份 weekly_report_helper 数据库
        const dbPath = '/home/zhangyufeng/.openclaw/memory/weekly_report_helper.sqlite';
        if (fs.existsSync(dbPath)) {
            const dbBackupDir = path.join(tempBackupDir, 'weekly-report-db');
            fs.mkdirSync(dbBackupDir, { recursive: true });
            
            console.log(`[${new Date().toISOString()}] 备份 weekly_report_helper 数据库...`);
            
            const dbBackupPath = path.join(dbBackupDir, 'weekly_report_helper.sqlite');
            fs.copyFileSync(dbPath, dbBackupPath);
            
            backupManifest.files.push({
                name: 'weekly_report_helper.sqlite',
                path: 'weekly-report-db/weekly_report_helper.sqlite',
                size: fs.statSync(dbBackupPath).size
            });
        }
        
        // 5. 备份 Telegram 配置
        const telegramDir = '/home/zhangyufeng/.openclaw/telegram';
        if (fs.existsSync(telegramDir)) {
            const telegramBackupDir = path.join(tempBackupDir, 'telegram-config');
            fs.mkdirSync(telegramBackupDir, { recursive: true });
            
            console.log(`[${new Date().toISOString()}] 备份 Telegram 配置...`);
            
            const telegramBackupPath = path.join(telegramBackupDir, 'telegram-config.tar.gz');
            execSync(
                `tar -czf "${telegramBackupPath}" -C "${telegramDir}" .`,
                { stdio: 'inherit' }
            );
            
            backupManifest.files.push({
                name: 'telegram-config.tar.gz',
                path: 'telegram-config/telegram-config.tar.gz',
                size: fs.statSync(telegramBackupPath).size
            });
        }
        
        // 6. 保存备份清单
        const manifestPath = path.join(tempBackupDir, 'backup-manifest.json');
        fs.writeFileSync(manifestPath, JSON.stringify(backupManifest, null, 2));
        
        backupManifest.files.push({
            name: 'backup-manifest.json',
            path: 'backup-manifest.json',
            size: fs.statSync(manifestPath).size
        });
        
        // 7. 打包所有备份文件
        console.log(`[${new Date().toISOString()}] 打包所有备份文件...`);
        execSync(
            `tar -czf "${tarPath}" -C "${tempBackupDir}" .`,
            { stdio: 'inherit' }
        );
        
        console.log(`[${new Date().toISOString()}] 打包完成: ${tarPath}`);
        
        // 清理临时目录
        fs.rmSync(tempBackupDir, { recursive: true, force: true });
        
        // 上传到COS
        console.log(`[${new Date().toISOString()}] 上传到COS...`);
        
        const tarFileContent = fs.readFileSync(tarPath);
        const cosKey = `backups/${dateStr}/${tarFileName}`;
        
        await new Promise((resolve, reject) => {
            cos.putObject({
                Bucket: config.Bucket,
                Region: config.Region,
                Key: cosKey,
                Body: tarFileContent,
                ContentLength: tarFileContent.length
            }, (err, data) => {
                if (err) {
                    reject(err);
                } else {
                    resolve(data);
                }
            });
        });
        
        console.log(`[${new Date().toISOString()}] 上传成功: cos://${config.Bucket}/${cosKey}`);
        
        // 生成备份报告
        const totalSize = backupManifest.files.reduce((sum, file) => sum + file.size, 0);
        const report = {
            timestamp: new Date().toISOString(),
            backup: {
                fileName: tarFileName,
                cosKey: cosKey,
                size: tarFileContent.length,
                date: dateStr,
                time: timeStr,
                components: backupManifest.components,
                fileCount: backupManifest.files.length,
                totalSize: totalSize
            },
            tokenStats: tokenStats.success ? {
                usdCost: tokenStats.usdCost,
                cnyCost: tokenStats.cnyCost,
                included: true
            } : {
                included: false,
                error: "Token统计失败"
            },
            manifest: {
                files: backupManifest.files.map(file => ({
                    name: file.name,
                    size: file.size,
                    path: file.path
                }))
            }
        };
        
        // 保存备份报告
        const reportDir = path.join(WORKSPACE_DIR, 'backup-reports');
        if (!fs.existsSync(reportDir)) {
            fs.mkdirSync(reportDir, { recursive: true });
        }
        
        const reportFile = path.join(reportDir, `backup-report-enhanced-${dateStr}-${timestamp}.json`);
        fs.writeFileSync(reportFile, JSON.stringify(report, null, 2));
        console.log(`[${new Date().toISOString()}] 备份报告已保存: ${reportFile}`);
        
        // 输出汇总信息
        console.log(`\n📊 增强版备份完成汇总:`);
        console.log(`📅 日期: ${dateStr} ${timeStr}`);
        console.log(`📁 文件: ${tarFileName}`);
        console.log(`💾 大小: ${(tarFileContent.length / 1024 / 1024).toFixed(2)} MB`);
        console.log(`📦 组件: ${Object.keys(backupManifest.components).length} 个`);
        console.log(`📄 文件数: ${backupManifest.files.length} 个`);
        console.log(`🌐 位置: cos://${config.Bucket}/${cosKey}`);
        
        // 输出组件详情
        console.log(`\n🔧 备份组件详情:`);
        console.log(`├─ 主工作空间: ${WORKSPACE_DIR}`);
        console.log(`├─ weekly_report_helper 工作空间: ${backupManifest.components.weekly_report_helper.workspace}`);
        console.log(`├─ weekly_report_helper 代理配置: ${backupManifest.components.weekly_report_helper.agents}`);
        console.log(`├─ weekly_report_helper 数据库: ${backupManifest.components.weekly_report_helper.memory}`);
        console.log(`└─ Telegram 配置: ${backupManifest.components.telegram_config}`);
        
        if (tokenStats.success) {
            console.log(`\n💰 Token费用统计:`);
            console.log(`└─ 估算费用: $${tokenStats.usdCost} (约 ${tokenStats.cnyCost} 元)`);
        }
        
        // 清理临时文件
        fs.unlinkSync(tarPath);
        console.log(`\n[${new Date().toISOString()}] 增强版备份完成！`);
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 增强版备份失败:`, error.message);
        
        // 清理临时目录（如果存在）
        const tempBackupDir = path.join(BACKUP_DIR, 'temp-enhanced-backup');
        if (fs.existsSync(tempBackupDir)) {
            fs.rmSync(tempBackupDir, { recursive: true, force: true });
        }
        
        // 清理临时文件（如果存在）
        if (fs.existsSync(tarPath)) {
            fs.unlinkSync(tarPath);
        }
        
        const errorMsg = `⚠️ 增强版备份失败 (${dateStr} ${timeStr})\n错误: ${error.message}`;
        const errorFile = `/home/zhangyufeng/.openclaw/workspace/backup-reports/error-enhanced-${Date.now()}.txt`;
        fs.writeFileSync(errorFile, errorMsg);
        
        process.exit(1);
    }
}

// 主函数
async function main() {
    try {
        await backup();
        process.exit(0);
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 备份主函数失败:`, error);
        process.exit(1);
    }
}

// 如果是直接运行
if (require.main === module) {
    main();
}

module.exports = {
    backup,
    runTokenStats
};
