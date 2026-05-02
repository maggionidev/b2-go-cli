package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func menu(client *s3.Client, cfg Config) {
	var buckets []string

	for {
		fmt.Println()
		fmt.Println(panelStyle.Render(
			bold.Render("B2 Manager") + "  " + dim.Render("v1.0") + "\n\n" +
				"  " + cyan.Render("[1]") + "  📦 Listar buckets\n" +
				"  " + cyan.Render("[2]") + "  📂 Ver arquivos de um bucket\n" +
				"  " + cyan.Render("[3]") + "  ⬆  Upload de imagem\n" +
				"  " + cyan.Render("[4]") + "  🚪 Sair",
		))

		opcao := prompt("\n  Escolha")

		switch opcao {
		case "1":
			var err error
			buckets, err = listarBuckets(client)
			if err != nil {
				fmt.Println(red.Render("  ✗ Erro: ") + err.Error())
			}

		case "2":
			if len(buckets) == 0 {
				var err error
				buckets, err = listarBuckets(client)
				if err != nil || len(buckets) == 0 {
					continue
				}
			}
			nome := selecionarBucket(buckets)
			if err := listarArquivos(client, nome); err != nil {
				fmt.Println(red.Render("  ✗ Erro: ") + err.Error())
			}

		case "3":
			if len(buckets) == 0 {
				var err error
				buckets, err = listarBuckets(client)
				if err != nil || len(buckets) == 0 {
					continue
				}
			}
			if err := fazerUpload(client, cfg, buckets); err != nil {
				fmt.Println(red.Render("  ✗ Erro: ") + err.Error())
			}

		case "4":
			fmt.Println(dim.Render("\n  Até logo! 👋\n"))
			return

		default:
			fmt.Println(yellow.Render("  Opção inválida."))
		}
	}
}