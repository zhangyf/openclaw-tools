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
	Clients  map[string]string   `json:"clients"`   // 客户名 -> 润色后的内容
	RawData  map[string][]string `json:"raw_data"`  // 客户名 -> 原始内容列表（用于审计）
	FilePath string              `json:"file_path"`
}

// TextPolisher 文本润色器接口
type TextPolisher interface {
	Polish(text string, targetLength int) (string, error)
}

// COSClient COS客户端接口
type COSClient interface {
	DownloadFile(ctx context.Context, key string) ([]byte, error)
	UploadFile(ctx context.Context, key string, data []byte) error
	FileExists(ctx context.Context, key string) (bool, error)
	ListFiles(ctx context.Context, prefix string) ([]string, error)
}