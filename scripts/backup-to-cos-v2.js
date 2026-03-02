#!/usr/bin/env node
/**
 * OpenClaw 完整备份脚本
 * 备份OpenClaw配置 + workspace，确保可完全恢复
 */

const COS = require('cos-nodejs-sdk-v5');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// 自动加载.env文件
function loadEnv() {
    if (process.env.TENCENT_COS_SECRET_ID && process.env.TENCENT_COS_SECRET_KEY) {
        return;
    }
    
    // 尝试多个.env文件位置
    const envPaths = [
        path.join(process.env.HOME, '.openclaw/.env'),
        path.join(process.env.HOME, '.openclaw/workspace/.env'),
        path.join(process.env.HOME, '.env')
    ];
    
    for (const envPath of envPaths) {
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
            console.log(`[${new Date().toISOString()}] 已从 ${envPath} 加载环境变量`);
            break;
        }
    }
}

// 加载环境变量
loadEnv();

// COS配置
const config = {
    Bucket: 'openclaw-bakup-1251036673',
    Region: 'ap-singapore',
    SecretId: process.env.TENCENT_COS_SECRET_ID,
    SecretKey: process.env.TENCENT_COS_SECRET_KEY
};

if (!config.SecretId || !config.SecretKey) {
    console.error(`[${new Date().toISOString()}] 错误: 缺少COS密钥`);
    console.error(`[${new Date().toISOString()}] 请设置 TENCENT_COS_SECRET_ID 和 TENCENT_COS_SECRET_KEY`);
    process.exit(1);
}

const cos = new COS({
    SecretId: config.SecretId,
    SecretKey: config.SecretKey
});

// 关键备份路径
const BACKUP_PATHS = [
    {
        path: '/home/zhangyufeng/.openclaw',
        name: 'openclaw-config',
        exclude: ['cache', 'logs', 'node_modules', '.git']
    },
    {
        path: '/home/zhangyufeng/.openclaw/workspace',
        name: 'openclaw-workspace',
        exclude: ['node_modules', '.git']
    }
];

const BACKUP_DIR = '/tmp/openclaw-full-backup';

async function createFullBackup() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const dateStr = new Date().toISOString().split('T')[0];
    
    console.log(`[${new Date().toISOString()}] 开始完整备份...`);
    
    // 创建临时备份目录
    if (!fs.existsSync(BACKUP_DIR)) {
        fs.mkdirSync(BACKUP_DIR, { recursive: true });
    }
    
    const tarFileName = `openclaw-full-backup-${dateStr}-${timestamp}.tar.gz`;
    const tarPath = path.join(BACKUP_DIR, tarFileName);
    
    try {
        // 创建备份清单文件
        const manifest = {
            timestamp: new Date().toISOString(),
            version: '2.0',
            description: 'OpenClaw完整备份（配置+workspace）',
            paths: BACKUP_PATHS.map(p => ({
                path: p.path,
                name: p.name,
                exclude: p.exclude
            })),
            restore_instructions: [
                '1. 解压备份文件: tar -xzf backup.tar.gz -C /tmp/backup',
                '2. 恢复配置: cp -r /tmp/backup/openclaw-config ~/.openclaw',
                '3. 恢复workspace: cp -r /tmp/backup/openclaw-workspace ~/.openclaw/workspace',
                '4. 重启OpenClaw: openclaw gateway restart'
            ]
        };
        
        const manifestPath = path.join(BACKUP_DIR, 'manifest.json');
        fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));
        
        // 为每个路径创建tar包，然后打包到一起
        const tempTarDir = path.join(BACKUP_DIR, 'tars');
        if (!fs.existsSync(tempTarDir)) {
            fs.mkdirSync(tempTarDir, { recursive: true });
        }
        
        const tarFiles = [];
        
        for (const backupPath of BACKUP_PATHS) {
            console.log(`[${new Date().toISOString()}] 备份: ${backupPath.path}`);
            
            const tarName = `${backupPath.name}.tar.gz`;
            const tarFilePath = path.join(tempTarDir, tarName);
            
            // 构建排除参数
            const excludeArgs = backupPath.exclude.map(pattern => `--exclude='${pattern}'`).join(' ');
            
            // 打包目录
            execSync(
                `tar -czf "${tarFilePath}" -C "${backupPath.path}" ${excludeArgs} .`,
                { stdio: 'pipe' }
            );
            
            tarFiles.push(tarFilePath);
            console.log(`[${new Date().toISOString()}] 已打包: ${backupPath.name} (${fs.statSync(tarFilePath).size} bytes)`);
        }
        
        // 将所有tar文件和清单打包成最终备份
        console.log(`[${new Date().toISOString()}] 创建最终备份包...`);
        
        const allFiles = [...tarFiles, manifestPath];
        const fileList = allFiles.map(f => `"${f}"`).join(' ');
        
        execSync(
            `tar -czf "${tarPath}" -C "${BACKUP_DIR}" ${fileList.replace(BACKUP_DIR + '/', '')}`,
            { stdio: 'pipe' }
        );
        
        console.log(`[${new Date().toISOString()}] 最终备份包: ${tarPath} (${fs.statSync(tarPath).size} bytes)`);
        
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
        
        // 列出COS中的备份文件
        console.log(`[${new Date().toISOString()}] 检查现有备份...`);
        await new Promise((resolve, reject) => {
            cos.getBucket({
                Bucket: config.Bucket,
                Region: config.Region,
                Prefix: 'backups/'
            }, (err, data) => {
                if (err) {
                    console.error(`[${new Date().toISOString()}] 列出备份失败:`, err.message);
                } else {
                    const backups = data.Contents || [];
                    console.log(`[${new Date().toISOString()}] COS中现有备份: ${backups.length} 个`);
                    backups.slice(-5).forEach(item => {
                        console.log(`  - ${item.Key} (${(item.Size / 1024).toFixed(2)} KB)`);
                    });
                }
                resolve();
            });
        });
        
        // 清理临时文件
        allFiles.forEach(f => {
            if (fs.existsSync(f)) fs.unlinkSync(f);
        });
        if (fs.existsSync(tempTarDir)) fs.rmdirSync(tempTarDir);
        if (fs.existsSync(tarPath)) fs.unlinkSync(tarPath);
        
        console.log(`[${new Date().toISOString()}] 完整备份完成！`);
        console.log(`[${new Date().toISOString()}] 恢复说明:`);
        console.log(`  1. 下载备份文件`);
        console.log(`  2. tar -xzf ${tarFileName}`);
        console.log(`  3. 按照 manifest.json 中的说明恢复`);
        
    } catch (error) {
        console.error(`[${new Date().toISOString()}] 备份失败:`, error.message);
        process.exit(1);
    }
}

createFullBackup();