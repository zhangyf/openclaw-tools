#!/usr/bin/env python3
"""COS预签名URL生成 - 使用cos_python_sdk_v5生成临时下载链接"""
import sys, os
from qcloud_cos import CosConfig, CosS3Client

bucket = "openclaw-backup-tx-1251036673"
region = "ap-beijing"

secret_id = os.environ.get("TENCENT_COS_SECRET_ID")
secret_key = os.environ.get("TENCENT_COS_SECRET_KEY")
if not secret_id or not secret_key:
    print("COS密钥未设置", file=sys.stderr)
    sys.exit(1)

config = CosConfig(Region=region, SecretId=secret_id, SecretKey=secret_key)
client = CosS3Client(config)

cos_path = sys.argv[1]
expire_seconds = int(sys.argv[2]) if len(sys.argv) > 2 else 3600  # default 1h

try:
    url = client.get_presigned_download_url(
        Bucket=bucket,
        Key=cos_path,
        Expired=expire_seconds
    )
    print(url)
except Exception as e:
    print(f"生成预签名URL失败: {e}", file=sys.stderr)
    sys.exit(1)
