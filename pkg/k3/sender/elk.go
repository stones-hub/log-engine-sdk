package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"log-engine-sdk/pkg/k3"
	"log-engine-sdk/pkg/k3/config"
	"log-engine-sdk/pkg/k3/protocol"
)

type ELKServer struct {
	config elasticsearch.Config
	client *elasticsearch.Client
}

func NewELKServer(address []string, username, password, apikey string) (*ELKServer, error) {
	{
		// TODO 方便测试，ELK还未搭建
		return &ELKServer{
			config: elasticsearch.Config{},
			client: nil,
		}, nil
	}

	return NewELKServerWithConfig(config.ELK{
		Address:  address,
		Username: username,
		Password: password,
		ApiKey:   apikey,
	})
}

func NewELKServerWithConfig(elkServerConfig config.ELK) (*ELKServer, error) {
	var (
		cfg    elasticsearch.Config
		client *elasticsearch.Client
		err    error
	)

	cfg = elasticsearch.Config{
		Addresses: elkServerConfig.Address,
		Username:  elkServerConfig.Username,
		Password:  elkServerConfig.Password,
		APIKey:    elkServerConfig.ApiKey,
	}

	if client, err = elasticsearch.NewClient(cfg); err != nil {
		k3.K3LogError("Failed to create Elasticsearch client: %v", err)
		return nil, err
	}

	return &ELKServer{
		config: cfg,
		client: client,
	}, nil
}

func (e *ELKServer) Send(data []protocol.Data) error {

	for _, d := range data {
		if err := config.GlobalConsumer.Add(d); err != nil {
			k3.K3LogError("Failed to add data to consumer: %v", err)
		}
	}

	return nil

	/*
		var (
			err         error
			bulkData    []byte
			bulkRequest esapi.BulkRequest
			msg         string
			res         *esapi.Response
			body        []byte
		)

		// TODO 方便测试，ELK还未搭建
		// 批量插入数据
		if bulkData, err = e.prepareBulkData(data); err != nil {
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

	*/
}

func (e *ELKServer) prepareBulkData(data []protocol.Data) ([]byte, error) {
	var bulkData bytes.Buffer

	for _, d := range data {
		jsonData, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}

		bulkData.WriteString(fmt.Sprintf(`{"index":{"_id":"%s"}}\n`, d.UUID))
		bulkData.Write(jsonData)
		bulkData.WriteString("\n")
	}
	return bulkData.Bytes(), nil
}
