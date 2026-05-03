package main

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/schollz/progressbar/v3"
)

// ─────────────────────────────────────────────
// RESULTADO DE UPLOAD
// ─────────────────────────────────────────────

type resultadoUpload struct {
	RemotePath string
	URL        string
	Tamanho    int64
	Err        error
}

// ─────────────────────────────────────────────
// HELPERS DE URL E PATH
// ─────────────────────────────────────────────

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
// COLETA DE ARQUIVOS (arquivo único ou pasta)
// ─────────────────────────────────────────────

type arquivoParaEnviar struct {
	localPath  string // caminho absoluto local
	remotePath string // chave remota no bucket
}

// coletarArquivos retorna a lista de arquivos a enviar.
//
// Se localPath for um arquivo, retorna esse único arquivo com
// remotePath = destino + basename.
//
// Se localPath for uma pasta, percorre recursivamente preservando
// a estrutura interna relativa à raiz da pasta fornecida.
// Exemplo: pasta "fotos/" + destino "blog/" →
//   fotos/capa.jpg     → blog/capa.jpg
//   fotos/2024/a.png   → blog/2024/a.png
func coletarArquivos(localPath, destino string) ([]arquivoParaEnviar, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("caminho não encontrado: %w", err)
	}

	// Normaliza destino: sempre termina com "/"
	destino = strings.Trim(destino, "/")
	if destino != "" {
		destino += "/"
	}

	// Arquivo único
	if !info.IsDir() {
		return []arquivoParaEnviar{
			{
				localPath:  localPath,
				remotePath: destino + filepath.Base(localPath),
			},
		}, nil
	}

	// Pasta: percorre recursivamente
	raiz := filepath.Clean(localPath)
	var lista []arquivoParaEnviar

	err = filepath.WalkDir(raiz, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // pula diretórios, só processa arquivos
		}

		// Caminho relativo à raiz da pasta fornecida
		rel, err := filepath.Rel(raiz, path)
		if err != nil {
			return err
		}

		// Converte separador do OS para "/" (necessário no Windows)
		rel = filepath.ToSlash(rel)

		lista = append(lista, arquivoParaEnviar{
			localPath:  path,
			remotePath: destino + rel,
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("erro ao percorrer pasta: %w", err)
	}
	if len(lista) == 0 {
		return nil, fmt.Errorf("nenhum arquivo encontrado em %q", localPath)
	}
	return lista, nil
}

// ─────────────────────────────────────────────
// UPLOAD DE UM ÚNICO ARQUIVO (núcleo reutilizável)
// ─────────────────────────────────────────────

func enviarArquivo(
	client *s3.Client,
	cfg Config,
	bucket string,
	arq arquivoParaEnviar,
	mostrarBarra bool,
) resultadoUpload {
	f, err := os.Open(arq.localPath)
	if err != nil {
		return resultadoUpload{RemotePath: arq.remotePath, Err: fmt.Errorf("abrir: %w", err)}
	}
	defer f.Close()

	info, _ := f.Stat()
	tamanho := info.Size()
	contentType := detectarContentType(arq.localPath)

	var body interface{ Read([]byte) (int, error) } = f

	if mostrarBarra {
		bar := progressbar.NewOptions64(tamanho,
			progressbar.OptionSetDescription(
				fmt.Sprintf("  %-40s", truncarString(filepath.Base(arq.localPath), 38)),
			),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerPadding: "░",
				BarStart:      "│",
				BarEnd:        "│",
			}),
			progressbar.OptionSetWidth(30),
			progressbar.OptionShowBytes(true),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetPredictTime(false),
		)
		r := progressbar.NewReader(f, bar)
		// PutObject precisa de io.Reader — usa &r que implementa a interface
		_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(arq.remotePath),
			Body:          &r,
			ContentType:   aws.String(contentType),
			ContentLength: aws.Int64(tamanho),
		})
	} else {
		_ = body
		_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(arq.remotePath),
			Body:          f,
			ContentType:   aws.String(contentType),
			ContentLength: aws.Int64(tamanho),
		})
	}

	if err != nil {
		return resultadoUpload{RemotePath: arq.remotePath, Tamanho: tamanho, Err: err}
	}

	return resultadoUpload{
		RemotePath: arq.remotePath,
		URL:        urlArquivo(cfg, bucket, arq.remotePath),
		Tamanho:    tamanho,
	}
}

// ─────────────────────────────────────────────
// SUMÁRIO DE RESULTADOS
// ─────────────────────────────────────────────

func imprimirSumario(resultados []resultadoUpload, cfg Config) {
	var ok, falhas int
	var totalBytes int64

	for _, r := range resultados {
		if r.Err != nil {
			falhas++
		} else {
			ok++
			totalBytes += r.Tamanho
		}
	}

	// Cabeçalho
	titulo := fmt.Sprintf("✓ %d enviado(s)  •  %s", ok, formatarTamanho(totalBytes))
	if falhas > 0 {
		titulo += fmt.Sprintf("  •  %s", red.Render(fmt.Sprintf("✗ %d falha(s)", falhas)))
	}
	fmt.Println("\n" + panelStyle.Render(green.Render(titulo)))
	fmt.Println()

	// Lista de sucessos
	for _, r := range resultados {
		if r.Err != nil {
			continue
		}
		label := dim.Render("URL Custom:") 
		if cfg.CustomDomain == "" {
			label = dim.Render("URL B2:   ")
		}
		fmt.Printf("  %s  %s\n", cyan.Render(r.RemotePath), dim.Render(formatarTamanho(r.Tamanho)))
		fmt.Printf("  %s %s\n\n", label, yellow.Render(r.URL))
	}

	// Lista de falhas
	if falhas > 0 {
		fmt.Println(errorPanel.Render(red.Render("Falhas:")))
		for _, r := range resultados {
			if r.Err == nil {
				continue
			}
			fmt.Printf("  %s  %s\n", red.Render("✗"), r.RemotePath)
			fmt.Printf("    %s\n\n", dim.Render(r.Err.Error()))
		}
	}
}

// ─────────────────────────────────────────────
// UPLOAD INTERATIVO (menu)
// ─────────────────────────────────────────────

func fazerUpload(client *s3.Client, cfg Config, buckets []string) error {
	fmt.Println(panelStyle.Render(
		bold.Render("⬆  Upload") + "\n" +
			dim.Render("Arquivo único ou pasta inteira."),
	))

	// 1. Bucket
	fmt.Printf("\n  Buckets: %s\n", strings.Join(bucketsCyan(buckets), "  "))
	bucketNome := selecionarBucket(buckets)

	// 2. Pasta de destino remota
	pasta := prompt("  Pasta de destino (enter para raiz)")
	pasta = strings.Trim(pasta, "/")
	if pasta != "" {
		pasta += "/"
	}

	// 3. Caminho local (arquivo ou pasta)
	var caminhoLocal string
	for {
		caminhoLocal = prompt("  Caminho local (arquivo ou pasta)")
		if _, err := os.Stat(caminhoLocal); err == nil {
			break
		}
		fmt.Println(red.Render("  ✗ Caminho não encontrado. Tente novamente."))
	}

	// 4. Coletar arquivos
	arquivos, err := coletarArquivos(caminhoLocal, pasta)
	if err != nil {
		return err
	}

	info, _ := os.Stat(caminhoLocal)
	ehPasta := info.IsDir()

	// Para arquivo único: permite alterar o nome remoto
	if !ehPasta {
		nomeDefault := arquivos[0].remotePath
		fmt.Printf("  Nome remoto [%s]: ", dim.Render(nomeDefault))
		if novo := lerLinha(); novo != "" {
			arquivos[0].remotePath = strings.TrimLeft(novo, "/")
		}
	} else {
		fmt.Printf("\n  %s  %s arquivo(s) encontrado(s)\n",
			cyan.Render("→"),
			bold.Render(fmt.Sprintf("%d", len(arquivos))),
		)
	}

	fmt.Println()

	// 5. Enviar
	resultados := make([]resultadoUpload, 0, len(arquivos))
	for _, arq := range arquivos {
		res := enviarArquivo(client, cfg, bucketNome, arq, true)
		if res.Err != nil {
			fmt.Printf("  %s %s — %s\n",
				red.Render("✗"),
				arq.remotePath,
				dim.Render(res.Err.Error()),
			)
		}
		resultados = append(resultados, res)
	}

	// 6. Sumário
	imprimirSumario(resultados, cfg)
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

	destino := strings.TrimLeft(remotePath, "/")
	// Garante que destino de pasta termine com "/"
	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		if !strings.HasSuffix(destino, "/") {
			destino += "/"
		}
	}

	arquivos, err := coletarArquivos(filePath, destino)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}

	var falhou bool
	for _, arq := range arquivos {
		res := enviarArquivo(client, cfg, bucket, arq, false)
		if res.Err != nil {
			fmt.Fprintf(os.Stderr, "erro em %s: %v\n", arq.remotePath, res.Err)
			falhou = true
			continue
		}
		fmt.Println(res.URL)
	}

	if falhou {
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────

func detectarContentType(filePath string) string {
	ct := mime.TypeByExtension(filepath.Ext(filePath))
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}

func truncarString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "…" + s[len(s)-(max-1):]
}