package main

import (
	"context"
	"fmt"

	"github.com/isucon/isucandar/agent"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agent, err := agent.NewAgent(agent.WithBaseURL("https://github.com/"))
	if err != nil {
		panic(err)
	}

	req, err := agent.GET("/")
	if err != nil {
		panic(err)
	}

	res, err := agent.Do(ctx, req)
	if err != nil {
		panic(err)
	}

	resources, err := agent.ProcessHTML(ctx, res, res.Body)
	if err != nil {
		panic(err)
	}

	for url, resource := range resources {
		fmt.Printf("%s: %s: %s\n", resource.InitiatorType, resource.Response.Header.Get("Content-Type"), url)
	}
}
