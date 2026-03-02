#!/usr/bin/env node
/**
 * 群聊前检查脚本
 * 确保称呼正确，避免错误
 */

const { extractYutianInfo, checkTextForWrongTitles } = require('./check-yutian-name.js');

function preChatCheck(message) {
    console.log('🔍 群聊前检查...');
    
    const yutianInfo = extractYutianInfo();
    const wrongTitles = checkTextForWrongTitles(message, yutianInfo);
    
    if (wrongTitles.length > 0) {
        console.log(`❌ 发现错误称呼: ${wrongTitles.join(', ')}`);
        console.log(`✅ 应改为: ${yutianInfo.correctTitle}`);
        
        // 自动纠正
        let correctedMessage = message;
        yutianInfo.wrongTitles.forEach(wrongTitle => {
            // 避免把"于田"中的"田"也替换了
            if (wrongTitle === '田' && message.includes('于田')) {
                // 特殊处理：不替换"于田"中的"田"
                console.log(`⚠️ 保留"于田"中的"田"`);
            } else {
                const regex = new RegExp(wrongTitle, 'g');
                correctedMessage = correctedMessage.replace(regex, yutianInfo.correctTitle);
            }
        });
        
        console.log(`原消息: ${message}`);
        console.log(`纠正后: ${correctedMessage}`);
        
        return {
            pass: false,
            wrongTitles,
            correctTitle: yutianInfo.correctTitle,
            correctedMessage,
            originalMessage: message
        };
    }
    
    console.log('✅ 称呼正确');
    return {
        pass: true,
        correctTitle: yutianInfo.correctTitle
    };
}

// 测试
if (require.main === module) {
    const testMessages = [
        '田女士，最近怎么样？',
        '于老师的孩子放假了吗？',
        '田小姐在美国还好吗？',
        '于田最近有联系吗？',
        '大家好，田女士也在群里啊'
    ];
    
    testMessages.forEach((msg, i) => {
        console.log(`\n--- 测试 ${i+1} ---`);
        const result = preChatCheck(msg);
        if (!result.pass) {
            console.log(`建议发送: ${result.correctedMessage}`);
        }
    });
}

module.exports = {
    preChatCheck
};