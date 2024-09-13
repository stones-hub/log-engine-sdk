package main

import "log-engine-sdk/pkg/k3/watch"

type Temp struct {
	Name string `json:"name,omitempty"`
	Age  int    `json:"age,omitempty"`
}

func main() {

	fPath := "/Users/yelei/data/code/go-projects/log-engine-sdk/example/logs"
	fStatePath := "/Users/yelei/data/code/go-projects/log-engine-sdk/example/logs/sate/state.json"
	watch.Run(fPath, fStatePath)

}
