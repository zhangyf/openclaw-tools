#!/usr/bin/env python3
"""COS上传工具 - 使用cos_python_sdk_v5上传文件"""
import sys, os
from qcloud_cos import CosConfig, CosS3Client

def upload(content, cos_path):
    secret_id = os.environ.get("TENCENT_COS_SECRET_ID")
    secret_key = os.environ.get("TENCENT_COS_SECRET_KEY")
    if not secret_id or not secret_key:
        print("COS密钥未设置", file=sys.stderr)
        return False

    config = CosConfig(
        Region="ap-beijing",
        SecretId=secret_id,
        SecretKey=secret_key,
        Token=None,
        Scheme="https"
    )
    client = CosS3Client(config)
    
    try:
        response = client.put_object(
            Bucket="openclaw-backup-tx-1251036673",
            Body=content.encode("utf-8"),
            Key=cos_path
        )
        print(f"  ☁️  已上传COS: openclaw-backup-tx-1251036673/{cos_path}")
        return True
    except Exception as e:
        print(f"  ⚠️  COS上传失败: {e}", file=sys.stderr)
        return False

if __name__ == "__main__":
    content = sys.stdin.read()
    cos_path = sys.argv[1] if len(sys.argv) > 1 else "test.txt"
    if not upload(content, cos_path):
        sys.exit(1)
