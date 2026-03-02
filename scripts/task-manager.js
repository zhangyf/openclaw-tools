#!/usr/bin/env node
/**
 * 任务管理系统
 * 创建、更新、检查任务状态
 */

const fs = require('fs');
const path = require('path');

const TASKS_DIR = '/home/zhangyufeng/.openclaw/workspace/tasks';
const ACTIVE_DIR = path.join(TASKS_DIR, 'active');
const COMPLETED_DIR = path.join(TASKS_DIR, 'completed');
const FAILED_DIR = path.join(TASKS_DIR, 'failed');
const INDEX_FILE = path.join(TASKS_DIR, 'tasks.json');

// 确保目录存在
[ACTIVE_DIR, COMPLETED_DIR, FAILED_DIR].forEach(dir => {
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
});

// 生成任务ID
function generateTaskId() {
    const timestamp = Date.now();
    const random = Math.floor(Math.random() * 1000);
    return `task-${timestamp}-${random}`;
}

// 创建新任务
function createTask(description, details, deadline, priority = 'medium', assignedBy = '老师') {
    const taskId = generateTaskId();
    const now = new Date();
    
    const task = {
        id: taskId,
        created: now.toISOString(),
        deadline: deadline, // ISO格式或"今天12:00"等
        priority: priority,
        status: 'active',
        description: description,
        details: details,
        assignedBy: assignedBy,
        progress: '待开始',
        checkpoints: [],
        lastUpdated: now.toISOString()
    };
    
    // 保存任务文件
    const taskFile = path.join(ACTIVE_DIR, `${taskId}.json`);
    fs.writeFileSync(taskFile, JSON.stringify(task, null, 2));
    
    // 更新索引
    updateIndex(task);
    
    console.log(`✅ 任务创建: ${taskId}`);
    console.log(`描述: ${description}`);
    console.log(`截止: ${deadline}`);
    console.log(`文件: ${taskFile}`);
    
    return taskId;
}

// 更新任务进度
function updateTask(taskId, updates) {
    const taskFile = path.join(ACTIVE_DIR, `${taskId}.json`);
    
    if (!fs.existsSync(taskFile)) {
        console.error(`❌ 任务不存在: ${taskId}`);
        return false;
    }
    
    const task = JSON.parse(fs.readFileSync(taskFile, 'utf8'));
    
    // 添加检查点
    if (updates.action) {
        task.checkpoints.push({
            time: new Date().toISOString(),
            action: updates.action,
            status: updates.status || 'in-progress'
        });
    }
    
    // 更新其他字段
    Object.assign(task, updates);
    task.lastUpdated = new Date().toISOString();
    
    fs.writeFileSync(taskFile, JSON.stringify(task, null, 2));
    updateIndex(task);
    
    console.log(`📝 任务更新: ${taskId}`);
    if (updates.progress) console.log(`进度: ${updates.progress}`);
    if (updates.action) console.log(`操作: ${updates.action}`);
    
    return true;
}

// 完成任务
function completeTask(taskId, completionNote = '任务完成') {
    const activeFile = path.join(ACTIVE_DIR, `${taskId}.json`);
    
    if (!fs.existsSync(activeFile)) {
        console.error(`❌ 任务不存在: ${taskId}`);
        return false;
    }
    
    const task = JSON.parse(fs.readFileSync(activeFile, 'utf8'));
    task.status = 'completed';
    task.completedAt = new Date().toISOString();
    task.completionNote = completionNote;
    
    // 移动到完成目录
    const completedFile = path.join(COMPLETED_DIR, `${taskId}.json`);
    fs.writeFileSync(completedFile, JSON.stringify(task, null, 2));
    fs.unlinkSync(activeFile);
    
    updateIndex(task, 'move');
    
    console.log(`🎉 任务完成: ${taskId}`);
    console.log(`描述: ${task.description}`);
    console.log(`完成时间: ${task.completedAt}`);
    
    return true;
}

// 标记任务失败
function failTask(taskId, failureReason = '任务失败') {
    const activeFile = path.join(ACTIVE_DIR, `${taskId}.json`);
    
    if (!fs.existsSync(activeFile)) {
        console.error(`❌ 任务不存在: ${taskId}`);
        return false;
    }
    
    const task = JSON.parse(fs.readFileSync(activeFile, 'utf8'));
    task.status = 'failed';
    task.failedAt = new Date().toISOString();
    task.failureReason = failureReason;
    
    // 移动到失败目录
    const failedFile = path.join(FAILED_DIR, `${taskId}.json`);
    fs.writeFileSync(failedFile, JSON.stringify(task, null, 2));
    fs.unlinkSync(activeFile);
    
    updateIndex(task, 'move');
    
    console.log(`❌ 任务失败: ${taskId}`);
    console.log(`描述: ${task.description}`);
    console.log(`原因: ${failureReason}`);
    
    return true;
}

// 更新索引
function updateIndex(task, action = 'update') {
    let index = {};
    if (fs.existsSync(INDEX_FILE)) {
        index = JSON.parse(fs.readFileSync(INDEX_FILE, 'utf8'));
    }
    
    if (action === 'move') {
        // 从active移动到其他状态，更新索引
        delete index[task.id];
    }
    
    index[task.id] = {
        id: task.id,
        status: task.status,
        description: task.description,
        deadline: task.deadline,
        priority: task.priority,
        lastUpdated: task.lastUpdated,
        file: task.status === 'active' ? 
            path.join('active', `${task.id}.json`) :
            path.join(task.status === 'completed' ? 'completed' : 'failed', `${task.id}.json`)
    };
    
    fs.writeFileSync(INDEX_FILE, JSON.stringify(index, null, 2));
}

// 列出任务
function listTasks(status = 'active') {
    let dir;
    switch(status) {
        case 'active': dir = ACTIVE_DIR; break;
        case 'completed': dir = COMPLETED_DIR; break;
        case 'failed': dir = FAILED_DIR; break;
        default: dir = ACTIVE_DIR;
    }
    
    console.log(`\n📋 ${status === 'active' ? '进行中' : status === 'completed' ? '已完成' : '失败'}任务列表:`);
    console.log('=' .repeat(50));
    
    if (!fs.existsSync(dir) || fs.readdirSync(dir).length === 0) {
        console.log('暂无任务');
        return [];
    }
    
    const tasks = [];
    fs.readdirSync(dir).forEach(file => {
        if (file.endsWith('.json')) {
            const taskPath = path.join(dir, file);
            const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
            tasks.push(task);
            
            const deadlineStr = task.deadline ? `截止: ${task.deadline}` : '无截止时间';
            const priorityIcon = task.priority === 'high' ? '🔥' : task.priority === 'medium' ? '⚠️' : '📌';
            
            console.log(`${priorityIcon} ${task.id}`);
            console.log(`  ${task.description}`);
            console.log(`  ${deadlineStr}`);
            console.log(`  进度: ${task.progress}`);
            console.log(`  最后更新: ${new Date(task.lastUpdated).toLocaleString('zh-CN')}`);
            console.log('');
        }
    });
    
    return tasks;
}

// 检查过期任务
function checkDeadlines() {
    console.log('\n⏰ 检查任务截止时间:');
    console.log('=' .repeat(50));
    
    const now = new Date();
    let expiredCount = 0;
    
    if (!fs.existsSync(ACTIVE_DIR)) return;
    
    fs.readdirSync(ACTIVE_DIR).forEach(file => {
        if (file.endsWith('.json')) {
            const taskPath = path.join(ACTIVE_DIR, file);
            const task = JSON.parse(fs.readFileSync(taskPath, 'utf8'));
            
            if (task.deadline) {
                const deadline = new Date(task.deadline);
                const hoursLeft = (deadline - now) / (1000 * 60 * 60);
                
                if (hoursLeft < 0) {
                    console.log(`❌ 已过期: ${task.id} - ${task.description}`);
                    console.log(`   应于: ${deadline.toLocaleString('zh-CN')}`);
                    expiredCount++;
                } else if (hoursLeft < 24) {
                    console.log(`⚠️ 即将到期 (${Math.ceil(hoursLeft)}小时): ${task.id} - ${task.description}`);
                }
            }
        }
    });
    
    if (expiredCount === 0) {
        console.log('✅ 无过期任务');
    }
    
    return expiredCount;
}

// CLI接口
if (require.main === module) {
    const args = process.argv.slice(2);
    const command = args[0];
    
    switch(command) {
        case 'create':
            if (args.length < 3) {
                console.log('用法: node task-manager.js create "描述" "详情" [截止时间] [优先级]');
                break;
            }
            createTask(args[1], args[2], args[3] || null, args[4] || 'medium');
            break;
            
        case 'update':
            if (args.length < 3) {
                console.log('用法: node task-manager.js update <taskId> "进度更新" [操作]');
                break;
            }
            updateTask(args[1], {
                progress: args[2],
                action: args[3] || null
            });
            break;
            
        case 'complete':
            if (args.length < 2) {
                console.log('用法: node task-manager.js complete <taskId> [完成说明]');
                break;
            }
            completeTask(args[1], args[2] || '任务完成');
            break;
            
        case 'fail':
            if (args.length < 3) {
                console.log('用法: node task-manager.js fail <taskId> "失败原因"');
                break;
            }
            failTask(args[1], args[2]);
            break;
            
        case 'list':
            listTasks(args[1] || 'active');
            break;
            
        case 'check':
            checkDeadlines();
            break;
            
        default:
            console.log('任务管理系统');
            console.log('命令:');
            console.log('  create <描述> <详情> [截止时间] [优先级] - 创建任务');
            console.log('  update <taskId> <进度> [操作] - 更新任务');
            console.log('  complete <taskId> [说明] - 完成任务');
            console.log('  fail <taskId> <原因> - 标记失败');
            console.log('  list [active|completed|failed] - 列出任务');
            console.log('  check - 检查截止时间');
            break;
    }
}

module.exports = {
    createTask,
    updateTask,
    completeTask,
    failTask,
    listTasks,
    checkDeadlines
};