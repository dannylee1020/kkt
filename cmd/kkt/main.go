package main

import (
	"fmt"
	"os"

	"github.com/dannylee1020/kkt/internal/workflow"
)

func main() {
	if err := workflow.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
