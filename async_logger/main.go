package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := StartMyMicroservice(ctx, ":8080", `{"test": ["/main.Biz/*"]}`)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("server started on :8080")
	<-ctx.Done()
	fmt.Println("shutting down")
}
