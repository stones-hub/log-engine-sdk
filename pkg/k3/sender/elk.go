package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"io"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/protocol"
)

// TODO 目前只做一个伪装实现，后续补充，方便测试

type ELKServer struct {
	config elasticsearch.Config
	client *elasticsearch.Client
}

// TODO 后续根据情况调整
func NewELKServerWithConfig() (*ELKServer, error) {
	var (
		elkConfig elasticsearch.Config
		elkClient *elasticsearch.Client
		err       error
	)

	elkConfig = elasticsearch.Config{
		Addresses: protocol.GlobalConfig.ELK.Addresses,
		Username:  protocol.GlobalConfig.ELK.Username,
		Password:  protocol.GlobalConfig.ELK.Password,
		APIKey:    protocol.GlobalConfig.ELK.APIKey,
	}

	if elkClient, err = elasticsearch.NewClient(elkConfig); err != nil {
		k3.K3LogError("Failed to create Elasticsearch client: %v", err)
		return nil, err
	}

	return &ELKServer{
		config: elkConfig,
		client: elkClient,
	}, nil
}

func (e *ELKServer) Send(data []protocol.Data) error {
	var (
		err         error
		bulkData    []byte
		bulkRequest esapi.BulkRequest
		msg         string
		res         *esapi.Response
		body        []byte
	)

	// 批量插入数据
	if bulkData, err = prepareBulkData(data); err != nil {
		msg = "Error preparing bulk data: " + err.Error()
		return errors.New(msg)
	}

	bulkRequest = esapi.BulkRequest{
		Index:                 "",
		Body:                  bytes.NewReader(bulkData),
		ListExecutedPipelines: nil,
		Pipeline:              "",
		Refresh:               "",
		RequireAlias:          nil,
		RequireDataStream:     nil,
		Routing:               "",
		Source:                nil,
		SourceExcludes:        nil,
		SourceIncludes:        nil,
		Timeout:               0,
		DocumentType:          "",
		WaitForActiveShards:   "",
		Pretty:                false,
		Human:                 false,
		ErrorTrace:            false,
		FilterPath:            nil,
		Header:                nil,
	}

	// 发送批量请求
	if res, err = bulkRequest.Do(context.Background(), e.client); err != nil {
		return err
	}

	defer res.Body.Close()

	if res.IsError() {
		body, _ = io.ReadAll(res.Body)
		return errors.New(string(body))
	} else {
		k3.K3LogInfo("Document send successfully")
	}
	return nil
}

func prepareBulkData(data []protocol.Data) ([]byte, error) {
	var bulkData bytes.Buffer

	for _, d := range data {
		jsonData, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}

		// bulkData.WriteString(fmt.Sprintf(`{"index":{"_id":"%d"}}\n`, d.ID))
		bulkData.Write(jsonData)
		bulkData.WriteString("\n")
	}
	return bulkData.Bytes(), nil
}
