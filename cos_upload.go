// COS上传工具 — Go语言版本
// 用法：echo "内容" | go run cos_upload.go fortune/2026/05/2026-05-27.txt
// 从stdin读取内容，上传到腾讯云COS指定路径

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

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
		fmt.Fprintln(os.Stderr, "用法: echo \"内容\" | cos_upload <cos-path>")
		os.Exit(1)
	}
	cosPath := os.Args[1]

	secretID := getEnvOrExit("TENCENT_COS_SECRET_ID")
	secretKey := getEnvOrExit("TENCENT_COS_SECRET_KEY")

	// 读取stdin内容
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取输入失败: %v\n", err)
		os.Exit(1)
	}

	u, _ := url.Parse(bucketURL)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	_, err = client.Object.Put(context.Background(), cosPath, strings.NewReader(string(content)), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "COS上传失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  ☁️  已上传COS: openclaw-backup-tx-1251036673/%s\n", cosPath)
}
