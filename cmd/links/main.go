package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jacekdobrowolski/goshort/services/links"
)

func main() {
	ctx := context.Background()
	if err := links.Run(ctx, os.Stdout, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
