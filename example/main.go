package main

import (
	"encoding/json"
	"fmt"
)

var jsonString = `{
  "name": "John Doe",
  "age": 30,
  "city": "New York",
	"address" : {
		"org":"china",
		"provide" :"guangzhou"
	}
}`

func main() {
	var (
		dataMap = make(map[string]interface{})
		err     error
	)
	if err = json.Unmarshal([]byte(jsonString), &dataMap); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(dataMap)
	}
}
