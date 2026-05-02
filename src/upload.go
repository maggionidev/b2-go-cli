package main

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/schollz/progressbar/v3"
)

// escaparCaminho escapa cada segmento de um path preservando as barras.
func escaparCaminho(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

// urlArquivo monta a URL pública de um objeto no bucket.
func urlArquivo(cfg Config, bucket, remotePath string) string {
	if cfg.CustomDomain != "" {
		domain := strings.TrimRight(cfg.CustomDomain, "/")
		return domain + "/" + escaparCaminho(strings.TrimLeft(remotePath, "/"))
	}
	return fmt.Sprintf("%s/file/%s/%s",
		strings.Replace(cfg.Endpoint, "s3.", "", 1),
		bucket,
		escaparCaminho(remotePath),
	)
}

// ─────────────────────────────────────────────
// UPLOAD INTERATIVO (menu)
// ─────────────────────────────────────────────

func fazerUpload(client *s3.Client, cfg Config, buckets []string) error {
	fmt.Println(panelStyle.Render(
		bold.Render("⬆  Upload de Imagem") + "\n" +
			dim.Render("Envie uma imagem para o bucket desejado."),
	))

	// 1. Bucket
	fmt.Printf("\n  Buckets: %s\n", strings.Join(bucketsCyan(buckets), "  "))
	bucketNome := selecionarBucket(buckets)

	// 2. Pasta de destino
	pasta := prompt("  Pasta de destino (enter para raiz)")
	pasta = strings.Trim(pasta, "/")
	if pasta != "" {
		pasta += "/"
	}

	// 3. Arquivo local
	var caminhoLocal string
	for {
		caminhoLocal = prompt("  Caminho local da imagem")
		if _, err := os.Stat(caminhoLocal); err == nil {
			break
		}
		fmt.Println(red.Render("  ✗ Arquivo não encontrado. Tente novamente."))
	}

	// 4. Nome remoto
	nomeRemotoDefault := pasta + filepath.Base(caminhoLocal)
	fmt.Printf("  Nome remoto [%s]: ", dim.Render(nomeRemotoDefault))
	nomeRemoto := lerLinha()
	if nomeRemoto == "" {
		nomeRemoto = nomeRemotoDefault
	}

	// 5. Content-type
	contentType := detectarContentType(caminhoLocal)

	// 6. Abrir arquivo
	f, err := os.Open(caminhoLocal)
	if err != nil {
		return fmt.Errorf("não foi possível abrir o arquivo: %w", err)
	}
	defer f.Close()

	info, _ := f.Stat()
	tamanho := info.Size()

	// 7. Barra de progresso
	fmt.Println()
	bar := progressbar.NewOptions64(tamanho,
		progressbar.OptionSetDescription(fmt.Sprintf("  Enviando %s", filepath.Base(caminhoLocal))),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "│",
			BarEnd:        "│",
		}),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowBytes(true),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetPredictTime(false),
	)
	reader := progressbar.NewReader(f, bar)

	// 8. Upload
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucketNome),
		Key:           aws.String(nomeRemoto),
		Body:          &reader,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(tamanho),
	})
	if err != nil {
		fmt.Println(errorPanel.Render(
			red.Render("✗ Falha no upload\n\n") + err.Error(),
		))
		return nil
	}

	// 9. Resultado
	fileURL := urlArquivo(cfg, bucketNome, nomeRemoto)

	successMsg := green.Render("✓ Upload concluído com sucesso!\n\n") +
		fmt.Sprintf("%s  %s\n", dim.Render("Arquivo:"), cyan.Render(nomeRemoto)) +
		fmt.Sprintf("%s  %s\n", dim.Render("Tamanho:"), formatarTamanho(tamanho)) +
		fmt.Sprintf("%s  %s\n\n", dim.Render("Tipo:   "), dim.Render(contentType))

	if cfg.CustomDomain != "" {
		successMsg += dim.Render("URL Custom:\n") + cyan.Render(fileURL)
	} else {
		successMsg += dim.Render("URL B2:\n") + yellow.Render(fileURL)
	}

	fmt.Println(successPanel.Render(successMsg))
	return nil
}

// ─────────────────────────────────────────────
// UPLOAD CLI (não-interativo)
// ─────────────────────────────────────────────

func uploadCLI(cfg Config, client *s3.Client, bucket, filePath, remotePath string) {
	if bucket == "" || filePath == "" || remotePath == "" {
		fmt.Fprintln(os.Stderr, "uso: b2manager upload --bucket NAME --file PATH --to REMOTE")
		os.Exit(1)
	}

	remotePath = strings.TrimLeft(remotePath, "/")
	contentType := detectarContentType(filePath)

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao abrir arquivo:", err)
		os.Exit(1)
	}
	defer f.Close()

	info, _ := f.Stat()
	tamanho := info.Size()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(remotePath),
		Body:          f,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(tamanho),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro no upload:", err)
		os.Exit(1)
	}

	fmt.Println(urlArquivo(cfg, bucket, remotePath))
}

// ─────────────────────────────────────────────
// HELPER
// ─────────────────────────────────────────────

func detectarContentType(filePath string) string {
	ct := mime.TypeByExtension(filepath.Ext(filePath))
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}