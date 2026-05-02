package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func emojiArquivo(nome string) string {
	ext := strings.ToLower(filepath.Ext(nome))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".svg":
		return "🖼 "
	case ".mp4", ".mov", ".avi", ".mkv":
		return "🎬"
	case ".mp3", ".wav", ".flac", ".ogg":
		return "🎵"
	case ".pdf":
		return "📄"
	case ".doc", ".docx":
		return "📝"
	case ".zip", ".tar", ".gz", ".rar":
		return "🗜 "
	case ".json", ".yaml", ".yml", ".toml":
		return "📋"
	case ".go", ".py", ".js", ".ts", ".rs":
		return "💻"
	default:
		return "📎"
	}
}

func formatarTamanho(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}