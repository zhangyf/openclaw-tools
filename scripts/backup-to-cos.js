#!/usr/bin/env node
/**
 * OpenClaw Daily Backup Script
 * 备份配置和会话记忆到腾讯云COS
 */

const COS = require('cos-nodejs-sdk-v5');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// COS配置 - 从环境变量读取密钥
const config = {
    Bucket: 'openclaw-bakup-1251036673',
    Region: 'ap-singapore',
    SecretId: process.env.TENCENT_COS_SECRET_ID,
    SecretKey: process.env.TENCENT_COS_SECRET_KEY
};

if (!config.SecretId || !config.SecretKey) {
    console.error('错误: 请设置环境变量 TENCENT_COS_SECRET_ID 和 TENCENT_COS_SECRET_KEY');
    process.exit(1);
}

const cos = new COS({
    SecretId: config.SecretId,
    SecretKey: config.SecretKey
});

const WORKSPACE_DIR = '/home/zhangyufeng/.openclaw/workspace';
const BACKUP_DIR = '/tmp/openclaw-backup';

async function backup() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const dateStr = new Date().toISOString().split('T')[0];
    
    console.log(`[${new Date().toISOString()}] 开始备份...`);
    
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
        
        // 清理临时文件
        fs.unlinkSync(tarPath);
        console.log(`[${new Date().toISOString()}] 备份完成！`);
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 备份失败:`, error.message);
        process.exit(1);
    }
}

backup();
