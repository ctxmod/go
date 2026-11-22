package main

import (
	"os"

	"example.com/cliproj/tools/cliproj/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args))
}
