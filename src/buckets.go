package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ─────────────────────────────────────────────
// LISTAGEM DE BUCKETS
// ─────────────────────────────────────────────

func listarBuckets(client *s3.Client) ([]string, error) {
	out, err := client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	if len(out.Buckets) == 0 {
		fmt.Println(yellow.Render("  Nenhum bucket encontrado."))
		return nil, nil
	}

	fmt.Println()
	header := fmt.Sprintf("  %-4s  %-40s  %-20s",
		bold.Render("#"),
		magenta.Render("Nome"),
		bold.Render("Criado em"),
	)
	fmt.Println(header)
	fmt.Println("  " + strings.Repeat("─", 68))

	names := make([]string, 0, len(out.Buckets))
	for i, b := range out.Buckets {
		created := ""
		if b.CreationDate != nil {
			created = b.CreationDate.Format("02/01/2006")
		}
		row := fmt.Sprintf("  %-4s  %-40s  %-20s",
			cyan.Bold(true).Render(fmt.Sprintf("[%d]", i+1)),
			bold.Render(aws.ToString(b.Name)),
			dim.Render(created),
		)
		fmt.Println(row)
		names = append(names, aws.ToString(b.Name))
	}
	fmt.Println()
	return names, nil
}

// ─────────────────────────────────────────────
// LISTAGEM DE ARQUIVOS
// ─────────────────────────────────────────────

func listarArquivos(client *s3.Client, bucket string) error {
	fmt.Printf("\n  %s %s\n\n",
		cyan.Bold(true).Render("📂 Conteúdo de:"),
		bold.Render(bucket),
	)

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
	})

	pastas := []string{}
	arquivos := []types.Object{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, p := range page.CommonPrefixes {
			pastas = append(pastas, aws.ToString(p.Prefix))
		}
		arquivos = append(arquivos, page.Contents...)
	}

	if len(pastas) == 0 && len(arquivos) == 0 {
		fmt.Println(dim.Render("  Bucket vazio."))
		return nil
	}

	// Imprime pastas
	for _, p := range pastas {
		nome := strings.TrimSuffix(p, "/")
		fmt.Printf("  📁 %s/\n", yellow.Render(nome))

		// Lista arquivos dentro da pasta (até 20)
		sub, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
			Bucket:  aws.String(bucket),
			Prefix:  aws.String(p),
			MaxKeys: aws.Int32(20),
		})
		if err == nil {
			for _, obj := range sub.Contents {
				key := strings.TrimPrefix(aws.ToString(obj.Key), p)
				if key == "" {
					continue
				}
				emoji := emojiArquivo(key)
				size := formatarTamanho(aws.ToInt64(obj.Size))
				fmt.Printf("  │  %s %s  %s\n",
					emoji,
					green.Render(key),
					dim.Render(size),
				)
			}
			if aws.ToBool(sub.IsTruncated) {
				fmt.Println(dim.Render("  │  … (mais arquivos)"))
			}
		}
	}

	// Imprime arquivos na raiz
	for _, obj := range arquivos {
		key := aws.ToString(obj.Key)
		emoji := emojiArquivo(key)
		size := formatarTamanho(aws.ToInt64(obj.Size))
		data := ""
		if obj.LastModified != nil {
			data = obj.LastModified.Format("02/01/2006 15:04")
		}
		fmt.Printf("  %s %s  %s  %s\n",
			emoji,
			green.Render(key),
			dim.Render(size),
			dim.Render(data),
		)
	}

	fmt.Println()
	return nil
}

// ─────────────────────────────────────────────
// HELPER
// ─────────────────────────────────────────────

func bucketsCyan(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = cyan.Render(n)
	}
	return out
}