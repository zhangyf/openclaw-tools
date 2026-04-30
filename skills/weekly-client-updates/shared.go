package main

import (
	"context"
)

// ClientUpdate 表示单个客户更新
type ClientUpdate struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// WeeklyReport 表示周报文件结构
type WeeklyReport struct {
	Year     int                 `json:"year"`
	Week     int                 `json:"week"`
	Clients  map[string]string   `json:"clients"`   // 客户名 -> 原始内容
	FilePath string              `json:"file_path"`
}

// COSClient COS客户端接口
type COSClient interface {
	DownloadFile(ctx context.Context, key string) ([]byte, error)
	UploadFile(ctx context.Context, key string, data []byte) error
	FileExists(ctx context.Context, key string) (bool, error)
	ListFiles(ctx context.Context, prefix string) ([]string, error)
}
