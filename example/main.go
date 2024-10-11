package main

import (
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
)

var jsonString = `{
    "log_level": "INFO", 
    "host_name" : "ali-gnfx-api-sdk4-01",
    "_id": "8816c977-854e-11ef-917e-00163e346885",
    "trace_id": "f0713b0d0ea4b3b5a068a808e2f38aa",
    "domain" : "gnuser.3k.com",
    "http_code" : 200,
    "host_ip" : "192.168.3.130",
    "code_name": "api_sdk4",
    "client_ip": "34.12.10.1",
    "timestamp" : "2024-10-09T17:41:30.703011223+08:00",
    "org": "gnfx",
    "log_src": "default",
    "project" : "ywzx",
    "event_id" : 1001,
    "event_name" : "user_login",
    "extend_data": {
        "uid": "2029753648",
        "game_name" : "坦克前线",
        "amount" : 30,
        "currency" : "RMB",
        "language" : "Android",
        "version" : "10.7.2",
        "code" : 1001,
        "content" : {
            "time_local": "2024-10-09 15:40:03",
            "channel": "trace",
            "content": "[trace.server]接收请求POST http://gnuser.3k.com/v5/user/login",
            "context": {
                "log_type": "trace.server",
                "url": "http://gnuser.3k.com/v5/user/login",
                "method": "POST",
                "input": "p=nh2%252FcdpyD",
                "output": {"msg": "Success"},
                "cost_ms": 49,
                "start_time": "2024-10-09 15:39:23.644",
                "end_time": "2024-10-09 15:39:23.693"
            }
        }
    }
}`

func main() {
	var (
		log protocol.ElasticSearchData
		err error
		b   []byte
	)
	if err = json.Unmarshal([]byte(jsonString), &log); err != nil {
		fmt.Println(err.Error())
		return
	} else {
		fmt.Println(log)
		fmt.Println("---->")
		if b, err = json.Marshal(log); err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(string(b))

	}
}
