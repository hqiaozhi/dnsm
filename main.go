package main

import (
	"dnsm/cmd"
	"fmt"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
