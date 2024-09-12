package main

import (
	"fmt"
	"log-engine-sdk/pkg/k3"
)

func main() {
	files, err := k3.FetchDirectory("/Users/yelei/data/code/go-projects/log-engine-sdk", -1)
	fmt.Println(err)
	for _, file := range files {
		fmt.Println(file)
	}
}
