package main

import (
	"fmt"
	"os"
)

func main() {
	env := os.Getenv("ENV")
	fmt.Println(Bootstrap(env))
}
