#!/usr/bin/env node
/**
 * OpenClaw Daily Backup Script
 * 备份配置和会话记忆到腾讯云COS
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

async function backup() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const dateStr = new Date().toISOString().split('T')[0];
    const timeStr = new Date().toLocaleTimeString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false });
    
    console.log(`[${new Date().toISOString()}] 开始备份...`);
    
    // 运行 token 统计
    const tokenStats = await runTokenStats();
    
    // 创建临时备份目录
    if (!fs.existsSync(BACKUP_DIR)) {
        fs.mkdirSync(BACKUP_DIR, { recursive: true });
    }
    
    const tarFileName = `openclaw-backup-${dateStr}-${timestamp}.tar.gz`;
    const tarPath = path.join(BACKUP_DIR, tarFileName);
    
    try {
        // 打包工作目录（排除node_modules）
        console.log(`[${new Date().toISOString()}] 打包文件...`);
        execSync(
            `tar -czf "${tarPath}" -C "${WORKSPACE_DIR}" --exclude='node_modules' --exclude='.git' .`,
            { stdio: 'inherit' }
        );
        
        console.log(`[${new Date().toISOString()}] 打包完成: ${tarPath}`);
        
        // 上传到COS
        console.log(`[${new Date().toISOString()}] 上传到COS...`);
        
        const fileContent = fs.readFileSync(tarPath);
        const cosKey = `backups/${dateStr}/${tarFileName}`;
        
        await new Promise((resolve, reject) => {
            cos.putObject({
                Bucket: config.Bucket,
                Region: config.Region,
                Key: cosKey,
                Body: fileContent,
                ContentLength: fileContent.length
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
        const report = {
            timestamp: new Date().toISOString(),
            backup: {
                fileName: tarFileName,
                cosKey: cosKey,
                size: fileContent.length,
                date: dateStr,
                time: timeStr
            },
            tokenStats: tokenStats.success ? {
                usdCost: tokenStats.usdCost,
                cnyCost: tokenStats.cnyCost,
                included: true
            } : {
                included: false,
                error: "Token统计失败"
            }
        };
        
        // 保存备份报告
        const reportDir = path.join(WORKSPACE_DIR, 'backup-reports');
        if (!fs.existsSync(reportDir)) {
            fs.mkdirSync(reportDir, { recursive: true });
        }
        
        const reportFile = path.join(reportDir, `backup-report-${dateStr}-${timestamp}.json`);
        fs.writeFileSync(reportFile, JSON.stringify(report, null, 2));
        console.log(`[${new Date().toISOString()}] 备份报告已保存: ${reportFile}`);
        
        // 输出汇总信息
        console.log(`\n📊 备份完成汇总:`);
        console.log(`📅 日期: ${dateStr} ${timeStr}`);
        console.log(`📁 文件: ${tarFileName}`);
        console.log(`💾 大小: ${(fileContent.length / 1024 / 1024).toFixed(2)} MB`);
        console.log(`🌐 位置: cos://${config.Bucket}/${cosKey}`);
        
        if (tokenStats.success) {
            console.log(`💰 Token费用: $${tokenStats.usdCost} (约 ${tokenStats.cnyCost} 元)`);
        }
        
        // 清理临时文件
        fs.unlinkSync(tarPath);
        console.log(`[${new Date().toISOString()}] 备份完成！`);
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 备份失败:`, error.message);
        process.exit(1);
    }
}

backup();
