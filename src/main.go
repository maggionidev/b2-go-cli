package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Modo CLI: b2manager upload --bucket X --file Y --to Z
	if len(os.Args) > 1 && os.Args[1] == "upload" {
		fs := flag.NewFlagSet("upload", flag.ExitOnError)
		bucket   := fs.String("bucket", "", "nome do bucket")
		filePath := fs.String("file",   "", "caminho local do arquivo")
		to       := fs.String("to",     "", "caminho remoto de destino")
		_ = fs.Parse(os.Args[2:])

		cfg := carregarConfig()
		client, err := conectar(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro na conexão:", err)
			os.Exit(1)
		}
		uploadCLI(cfg, client, *bucket, *filePath, *to)
		return
	}

	// Modo interativo
	fmt.Println()
	fmt.Println(panelStyle.Render(
		bold.Render("  B2 Manager") + "  " + dim.Render("CLI para Backblaze B2  •  Go edition"),
	))
	fmt.Println()

	cfg := carregarConfig()

	client, err := conectar(cfg)
	if err != nil {
		fmt.Println(errorPanel.Render(
			red.Render("✗ Falha na conexão\n\n") + err.Error(),
		))
		os.Exit(1)
	}

	menu(client, cfg)
}