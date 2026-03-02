// backup-to-cos.go
// OpenClaw完整备份脚本（Go版本）
// 备份配置目录和workspace到腾讯云COS
// 确保可完全恢复

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/ini.v1"
)

// Config 备份配置
type Config struct {
	Bucket   string `json:"bucket"`
	Region   string `json:"region"`
	SecretID string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
}

// Manifest 备份清单
type Manifest struct {
	Timestamp       string   `json:"timestamp"`
	Version         string   `json:"version"`
	Description     string   `json:"description"`
	BackupPaths     []string `json:"backup_paths"`
	ExcludePatterns []string `json:"exclude_patterns"`
	RestoreSteps    []string `json:"restore_steps"`
}

// 常量定义
const (
	version     = "1.0.0"
	backupDir   = "/tmp/openclaw-go-backup"
	workspace   = "/home/zhangyufeng/.openclaw/workspace"
	configDir   = "/home/zhangyufeng/.openclaw"
)

func main() {
	log.Println("🚀 OpenClaw Go备份脚本启动")
	log.Printf("版本: %s", version)
	log.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))

	// 1. 加载配置
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 2. 创建备份目录
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatalf("❌ 创建备份目录失败: %v", err)
	}
	defer cleanup()

	// 3. 生成备份清单
	manifest := createManifest()
	manifestPath := filepath.Join(backupDir, "manifest.json")
	if err := saveManifest(manifest, manifestPath); err != nil {
		log.Fatalf("❌ 保存清单失败: %v", err)
	}

	// 4. 备份关键目录
	backupPaths := []struct {
		path    string
		name    string
		exclude []string
	}{
		{
			path: configDir,
			name: "openclaw-config",
			exclude: []string{
				"cache",
				"logs",
				"node_modules",
				".git",
			},
		},
		{
			path: workspace,
			name: "openclaw-workspace",
			exclude: []string{
				"node_modules",
				".git",
			},
		},
	}

	var tarFiles []string
	for _, bp := range backupPaths {
		log.Printf("📦 备份: %s (%s)", bp.name, bp.path)
		
		tarPath := filepath.Join(backupDir, fmt.Sprintf("%s.tar.gz", bp.name))
		if err := createTarGz(bp.path, tarPath, bp.exclude); err != nil {
			log.Fatalf("❌ 打包失败 %s: %v", bp.name, err)
		}
		
		// 检查文件大小
		if fi, err := os.Stat(tarPath); err == nil {
			log.Printf("  大小: %.2f MB", float64(fi.Size())/1024/1024)
		}
		
		tarFiles = append(tarFiles, tarPath)
	}

	// 5. 创建最终备份包
	finalBackupPath := filepath.Join(backupDir, 
		fmt.Sprintf("openclaw-backup-%s.tar.gz", 
			time.Now().Format("2006-01-02-150405")))
	
	if err := createFinalBackup(finalBackupPath, append(tarFiles, manifestPath)); err != nil {
		log.Fatalf("❌ 创建最终备份包失败: %v", err)
	}

	log.Printf("✅ 最终备份包: %s", finalBackupPath)

	// 6. 上传到COS
	if err := uploadToCOS(config, finalBackupPath); err != nil {
		log.Fatalf("❌ 上传到COS失败: %v", err)
	}

	log.Println("🎉 备份完成!")
}

// loadConfig 加载配置
func loadConfig() (*Config, error) {
	config := &Config{
		Bucket: "openclaw-bakup-1251036673",
		Region: "ap-singapore",
	}

	// 尝试从环境变量加载
	config.SecretID = os.Getenv("TENCENT_COS_SECRET_ID")
	config.SecretKey = os.Getenv("TENCENT_COS_SECRET_KEY")

	if config.SecretID != "" && config.SecretKey != "" {
		log.Println("✅ 从环境变量加载配置")
		return config, nil
	}

	// 尝试从.env文件加载
	envPaths := []string{
		filepath.Join(configDir, ".env"),
		filepath.Join(workspace, ".env"),
		filepath.Join(os.Getenv("HOME"), ".env"),
	}

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			log.Printf("📄 读取.env文件: %s", envPath)
			
			cfg, err := ini.Load(envPath)
			if err != nil {
				log.Printf("⚠️ 解析INI失败: %v", err)
				continue
			}

			section := cfg.Section("")
			if id := section.Key("TENCENT_COS_SECRET_ID").String(); id != "" {
				config.SecretID = id
			}
			if key := section.Key("TENCENT_COS_SECRET_KEY").String(); key != "" {
				config.SecretKey = key
			}

			if config.SecretID != "" && config.SecretKey != "" {
				log.Println("✅ 从.env文件加载配置成功")
				return config, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到有效的COS配置")
}

// createManifest 创建备份清单
func createManifest() *Manifest {
	return &Manifest{
		Timestamp:   time.Now().Format(time.RFC3339),
		Version:     version,
		Description: "OpenClaw完整备份（Go版本）",
		BackupPaths: []string{
			configDir,
			workspace,
		},
		ExcludePatterns: []string{
			"cache/",
			"logs/",
			"node_modules/",
			".git/",
		},
		RestoreSteps: []string{
			"1. 解压备份文件: tar -xzf backup.tar.gz -C /tmp/backup",
			"2. 恢复配置: cp -r /tmp/backup/openclaw-config ~/.openclaw",
			"3. 恢复workspace: cp -r /tmp/backup/openclaw-workspace ~/.openclaw/workspace",
			"4. 重启OpenClaw: openclaw gateway restart",
		},
	}
}

// saveManifest 保存清单文件
func saveManifest(manifest *Manifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// createTarGz 创建tar.gz压缩包
func createTarGz(sourceDir, outputPath string, exclude []string) error {
	// 使用系统tar命令（更可靠）
	args := []string{
		"-czf",
		outputPath,
		"-C",
		sourceDir,
		".",
	}

	// 添加排除参数
	for _, pattern := range exclude {
		args = append(args, "--exclude="+pattern)
	}

	cmd := exec.Command("tar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// createFinalBackup 创建最终备份包
func createFinalBackup(outputPath string, files []string) error {
	// 创建tar文件
	tarFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	// 创建gzip writer
	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 添加每个文件到tar
	for _, file := range files {
		if err := addFileToTar(tarWriter, file, backupDir); err != nil {
			return fmt.Errorf("添加文件 %s 失败: %v", file, err)
		}
	}

	return nil
}

// addFileToTar 添加文件到tar
func addFileToTar(tw *tar.Writer, filename, baseDir string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	// 创建tar header
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	// 计算相对路径
	relPath, err := filepath.Rel(baseDir, filename)
	if err != nil {
		return err
	}
	header.Name = relPath

	// 写入header
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// 写入文件内容
	_, err = io.Copy(tw, file)
	return err
}

// uploadToCOS 上传到腾讯云COS
func uploadToCOS(config *Config, filePath string) error {
	log.Println("☁️ 上传到腾讯云COS...")

	// 使用现有的Node.js脚本上传（保持兼容）
	// 未来可以替换为纯Go实现
	
	cmd := exec.Command("node", "/home/zhangyufeng/.openclaw/workspace/scripts/backup-to-cos-v2.js")
	cmd.Env = append(os.Environ(),
		"TENCENT_COS_SECRET_ID="+config.SecretID,
		"TENCENT_COS_SECRET_KEY="+config.SecretKey,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Node.js上传失败: %v\n输出: %s", err, output)
	}

	log.Println("✅ 上传成功（通过Node.js脚本）")
	return nil
}

// cleanup 清理临时文件
func cleanup() {
	log.Println("🧹 清理临时文件...")
	if err := os.RemoveAll(backupDir); err != nil {
		log.Printf("⚠️ 清理失败: %v", err)
	}
}