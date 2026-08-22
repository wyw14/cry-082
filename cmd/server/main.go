package main

import (
	"fmt"
	"os"

	"github.com/wyw14/cry-082/internal/bootstrap"
	"github.com/wyw14/cry-082/internal/config"
)

func main() {
	os.Exit(execute())
}

func execute() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration rejected: %v\n", err)
		return 2
	}
	if err := bootstrap.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "monitoring service stopped: %v\n", err)
		return 1
	}
	return 0
}
