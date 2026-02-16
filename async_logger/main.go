package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	const ACLData string = `{
		"logger1":          ["/main.Admin/Logging"],
		"logger2":          ["/main.Admin/Logging"],
		"stat1":            ["/main.Admin/Statistics"],
		"stat2":            ["/main.Admin/Statistics"],
		"biz_user":         ["/main.Biz/Check", "/main.Biz/Add"],
		"biz_admin":        ["/main.Biz/*"],
		"after_disconnect": ["/main.Biz/Add"]
}`

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := StartMyMicroservice(ctx, ":8080", ACLData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("server started on :8080")
	<-ctx.Done()
	fmt.Println("shutting down")
}
