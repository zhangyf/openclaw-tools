package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// RealCOSClient 实际的腾讯云COS客户端实现
type RealCOSClient struct {
	client *cos.Client
	bucket string
	region string
}

// NewRealCOSClient 创建新的COS客户端
func NewRealCOSClient(bucket, region, secretID, secretKey string) (*RealCOSClient, error) {
	// 从环境变量获取认证信息（如果参数为空）
	if secretID == "" {
		secretID = os.Getenv("WEEKLY_CLIENT_UPDATE_SECRET_ID")
	}
	if secretKey == "" {
		secretKey = os.Getenv("WEEKLY_CLIENT_UPDATE_SECRET_KEY")
	}
	if region == "" {
		region = "ap-beijing" // 默认北京区域
	}
	
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("腾讯云认证信息缺失，请提供secret-id和secret-key或设置WEEKLY_CLIENT_UPDATE_SECRET_ID/WEEKLY_CLIENT_UPDATE_SECRET_KEY环境变量")
	}
	
	if bucket == "" {
		return nil, fmt.Errorf("COS桶名称不能为空")
	}
	
	// 构建COS服务地址
	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	if err != nil {
		return nil, fmt.Errorf("解析COS URL失败: %v", err)
	}
	
	// 创建基础URL
	b := &cos.BaseURL{BucketURL: u}
	
	// 创建客户端
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
	
	return &RealCOSClient{
		client: client,
		bucket: bucket,
		region: region,
	}, nil
}

// DownloadFile 从COS下载文件
func (c *RealCOSClient) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	// 设置上下文超时
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	
	// 获取文件
	resp, err := c.client.Object.Get(ctx, key, nil)
	if err != nil {
		// 检查是否是文件不存在的错误
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NoSuchKey") {
			return nil, nil // 文件不存在，返回空
		}
		return nil, fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Body.Close()
	
	// 读取响应内容
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}
	
	return data, nil
}

// UploadFile 上传文件到COS
func (c *RealCOSClient) UploadFile(ctx context.Context, key string, data []byte) error {
	// 设置上下文超时
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	
	// 创建reader
	reader := strings.NewReader(string(data))
	
	// 上传文件
	_, err := c.client.Object.Put(ctx, key, reader, nil)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}
	
	return nil
}

// ListFiles 列出指定前缀的文件
func (c *RealCOSClient) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	
	var files []string
	var marker string
	isTruncated := true
	
	for isTruncated {
		opt := &cos.BucketGetOptions{
			Prefix:  prefix,
			Marker:  marker,
			MaxKeys: 1000,
		}
		
		v, _, err := c.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("列出文件失败: %v", err)
		}
		
		for _, content := range v.Contents {
			files = append(files, content.Key)
		}
		
		isTruncated = v.IsTruncated
		marker = v.NextMarker
	}
	
	return files, nil
}

// DeleteFile 删除COS上的文件
func (c *RealCOSClient) DeleteFile(ctx context.Context, key string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	
	_, err := c.client.Object.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	
	return nil
}

// FileExists 检查文件是否存在
func (c *RealCOSClient) FileExists(ctx context.Context, key string) (bool, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}
	
	_, err := c.client.Object.Head(ctx, key, nil)
	if err != nil {
		// 检查是否是404错误
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("检查文件存在失败: %v", err)
	}
	
	return true, nil
}

// GetFileInfo 获取文件信息
func (c *RealCOSClient) GetFileInfo(ctx context.Context, key string) (interface{}, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}
	
	resp, err := c.client.Object.Head(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}
	
	return resp, nil
}

// Helper functions

// EnsureCOSClient 确保使用真实的COS客户端
func EnsureCOSClient(cosClient COSClient, bucket, region, secretID, secretKey string) (*RealCOSClient, error) {
	// 如果已经是RealCOSClient，直接返回
	if realClient, ok := cosClient.(*RealCOSClient); ok {
		return realClient, nil
	}
	
	// 否则创建新的
	return NewRealCOSClient(bucket, region, secretID, secretKey)
}

// DefaultCOSConfig 从环境变量获取默认COS配置
func DefaultCOSConfig() (bucket, region, secretID, secretKey string) {
	bucket = os.Getenv("WEEKLY_CLIENT_UPDATE_BUCKET")
	region = os.Getenv("WEEKLY_CLIENT_UPDATE_REGION")
	if region == "" {
		region = "ap-beijing"
	}
	secretID = os.Getenv("WEEKLY_CLIENT_UPDATE_SECRET_ID")
	secretKey = os.Getenv("WEEKLY_CLIENT_UPDATE_SECRET_KEY")
	return
}