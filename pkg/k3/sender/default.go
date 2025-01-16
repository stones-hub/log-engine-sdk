package sender

import (
	"encoding/json"
	"fmt"
	"log-engine-sdk/pkg/k3/protocol"
)

type Default struct {
}

func (d *Default) Send(data []protocol.Data) error {
	var (
		b   []byte
		err error
	)
	if b, err = json.Marshal(data); err != nil {
		return err
	}
	fmt.Println("default sender : ", string(b))
	return nil
}

func (d *Default) Close() error {
	fmt.Println("default sender : ", "close default sender")
	return nil
}
