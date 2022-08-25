package main

import (
	"context"
	"fmt"
	"os"

	"github.com/110y/run"
	"github.com/110y/servergroup"
)

func main() {
	run.Run(server)
}

func server(ctx context.Context) int {
	var group servergroup.Group

	group.Add(&httpServer{})

	if err := group.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "the server has aborted: %s", err)
		return 1
	}

	return 0
}
