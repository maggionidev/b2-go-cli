package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var reader = bufio.NewReader(os.Stdin)

func lerLinha() string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func prompt(label string) string {
	fmt.Printf("%s: ", bold.Render(label))
	return lerLinha()
}

func promptSecret(label string) string {
	fmt.Printf("%s: ", bold.Render(label))
	b, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return strings.TrimSpace(string(b))
}

func selecionarBucket(buckets []string) string {
	for {
		fmt.Printf("%s: ", bold.Render("  Número do bucket"))
		val := lerLinha()
		n := 0
		_, err := fmt.Sscanf(val, "%d", &n)
		if err == nil && n >= 1 && n <= len(buckets) {
			return buckets[n-1]
		}
		fmt.Printf("%s  (1–%d)\n", red.Render("  ✗ Número inválido."), len(buckets))
	}
}