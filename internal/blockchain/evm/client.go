package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RPCClient struct {
	url        string
	httpClient *http.Client
}

func NewRPCClient(url string) (*RPCClient, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("rpc url is required")
	}
	return &RPCClient{
		url: url,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

func (c *RPCClient) call(ctx context.Context, method string, params any, out any) error {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc %s failed: code=%d message=%s", method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if out == nil {
		return nil
	}
	if len(rpcResp.Result) == 0 || string(rpcResp.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(rpcResp.Result, out); err != nil {
		return err
	}
	return nil
}

func (c *RPCClient) SendTransaction(ctx context.Context, from, to, data string) (string, error) {
	if from == "" || to == "" || data == "" {
		return "", fmt.Errorf("from, to and data are required")
	}
	params := []any{map[string]any{
		"from": from,
		"to":   to,
		"data": data,
	}}
	var txHash string
	if err := c.call(ctx, "eth_sendTransaction", params, &txHash); err != nil {
		return "", err
	}
	if txHash == "" {
		return "", fmt.Errorf("empty transaction hash")
	}
	return txHash, nil
}

func (c *RPCClient) BlockNumber(ctx context.Context) (uint64, error) {
	var raw string
	if err := c.call(ctx, "eth_blockNumber", []any{}, &raw); err != nil {
		return 0, err
	}
	if raw == "" {
		return 0, fmt.Errorf("empty block number")
	}
	return parseHexUint64(raw)
}

type Log struct {
	TxHash      string   `json:"transactionHash"`
	BlockNumber string   `json:"blockNumber"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
}

type DecodedLog struct {
	TxHash      string
	BlockNumber uint64
	Topics      []string
	Data        string
}

func (c *RPCClient) GetLogs(ctx context.Context, fromBlock, toBlock uint64, address string, topic0 []string) ([]DecodedLog, error) {
	filter := map[string]any{
		"fromBlock": fmt.Sprintf("0x%x", fromBlock),
		"toBlock":   fmt.Sprintf("0x%x", toBlock),
		"address":   address,
	}
	if len(topic0) > 0 {
		filter["topics"] = []any{topic0}
	}
	var raw []Log
	if err := c.call(ctx, "eth_getLogs", []any{filter}, &raw); err != nil {
		return nil, err
	}
	out := make([]DecodedLog, 0, len(raw))
	for _, item := range raw {
		block, err := parseHexUint64(item.BlockNumber)
		if err != nil {
			return nil, err
		}
		out = append(out, DecodedLog{
			TxHash:      item.TxHash,
			BlockNumber: block,
			Topics:      item.Topics,
			Data:        item.Data,
		})
	}
	return out, nil
}

type txReceipt struct {
	TransactionHash string `json:"transactionHash"`
	Status          string `json:"status"`
	BlockNumber     string `json:"blockNumber"`
}

type Receipt struct {
	TxHash      string
	Status      uint64
	BlockNumber uint64
}

func (c *RPCClient) GetTransactionReceipt(ctx context.Context, txHash string) (*Receipt, error) {
	var raw *txReceipt
	if err := c.call(ctx, "eth_getTransactionReceipt", []any{txHash}, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	status, err := parseHexUint64(raw.Status)
	if err != nil {
		return nil, err
	}
	block, err := parseHexUint64(raw.BlockNumber)
	if err != nil {
		return nil, err
	}
	return &Receipt{
		TxHash:      raw.TransactionHash,
		Status:      status,
		BlockNumber: block,
	}, nil
}

func parseHexUint64(value string) (uint64, error) {
	v := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if v == "" {
		return 0, nil
	}
	return strconv.ParseUint(v, 16, 64)
}
