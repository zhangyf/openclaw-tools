// COS预签名URL生成 — Go语言版本
// 用法：cos_presign fortune/2026/05/2026-05-27.txt [过期秒数]
// 默认过期时间：3600秒（1小时）

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

const bucketURL = "https://openclaw-backup-tx-1251036673.cos.ap-beijing.myqcloud.com"

func getEnvOrExit(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "错误: 环境变量 %s 未设置\n", key)
		os.Exit(1)
	}
	return v
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: cos_presign <cos-path> [过期秒数]")
		os.Exit(1)
	}
	cosPath := os.Args[1]
	expireSeconds := 3600 // 默认1小时
	if len(os.Args) >= 3 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil {
			expireSeconds = n
		}
	}

	secretID := getEnvOrExit("TENCENT_COS_SECRET_ID")
	secretKey := getEnvOrExit("TENCENT_COS_SECRET_KEY")

	u, _ := url.Parse(bucketURL)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	presignedURL, err := client.Object.GetPresignedURL(
		context.Background(),
		http.MethodGet,
		cosPath,
		secretID,
		secretKey,
		time.Duration(expireSeconds)*time.Second,
		nil,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成预签名URL失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(presignedURL.String())
}
