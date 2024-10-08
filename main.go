package main

import (
	"fmt"
	"os"

	cmd "github.com/kevineaton/d2tosite/cmd"
)

func main() {
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error: %+v\n", err.Error())
		os.Exit(1)
	}
}
