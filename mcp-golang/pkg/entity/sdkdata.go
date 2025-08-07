package entity

import (
	"encoding/json"
	"errors"
	"mcp-sdk/pkg/utils"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type MCPSdkBaseMsg struct {
	RequestID string `json:"request_id"`
	Endpoint  string `json:"endpoint"`
	Version   string `json:"version"`
	Method    string `json:"method"`
	Timestamp string `json:"ts"`
	Sign      string `json:"sign"`
}

type MCPSdkRequest struct {
	MCPSdkBaseMsg
	Request string `json:"request"`
}

func EmptyBridgeRequest(method string, version string) *MCPSdkRequest {
	return &MCPSdkRequest{
		MCPSdkBaseMsg: MCPSdkBaseMsg{
			Method:    method,
			Version:   version,
			Timestamp: strconv.FormatInt(time.Now().UnixMilli(), 10),
		},
	}
}

func (w *MCPSdkRequest) String() string {
	json, err := json.Marshal(w)
	if err != nil {
		return ""
	}
	return string(json)
}

func (w *MCPSdkRequest) DoSign(token string) (err error) {
	payload := make(map[string]string)
	payload["request_id"] = w.RequestID
	payload["endpoint"] = w.Endpoint
	payload["version"] = w.Version
	payload["method"] = w.Method
	payload["ts"] = w.Timestamp
	payload["request"] = w.Request

	signer := utils.NewWsDataSigner(payload, token, utils.AlgoSHA256)
	sign, err := signer.Sign()
	if err != nil {
		return err
	}
	w.Sign = sign
	return nil
}

func (w *MCPSdkRequest) DoVerify(token string) (ok bool, err error) {
	payload := make(map[string]string)
	payload["request_id"] = w.RequestID
	payload["endpoint"] = w.Endpoint
	payload["version"] = w.Version
	payload["method"] = w.Method
	payload["ts"] = w.Timestamp
	payload["request"] = w.Request

	signer := utils.NewWsDataSigner(payload, token, utils.AlgoSHA256)
	return signer.Verify(w.Sign)
}

type MCPSdkResponse struct {
	MCPSdkBaseMsg
	Response string `json:"response"`
}

func (w *MCPSdkResponse) String() string {
	json, err := json.Marshal(w)
	if err != nil {
		return ""
	}
	return string(json)
}

func (w *MCPSdkResponse) DoSign(token string) (err error) {
	payload := make(map[string]string)
	payload["request_id"] = w.RequestID
	payload["endpoint"] = w.Endpoint
	payload["version"] = w.Version
	payload["method"] = w.Method
	payload["ts"] = w.Timestamp
	payload["response"] = w.Response

	signer := utils.NewWsDataSigner(payload, token, utils.AlgoSHA256)
	sign, err := signer.Sign()
	if err != nil {
		return err
	}
	w.Sign = sign
	return nil
}

func (w *MCPSdkResponse) DoVerify(token string) (ok bool, err error) {
	payload := make(map[string]string)
	payload["request_id"] = w.RequestID
	payload["endpoint"] = w.Endpoint
	payload["version"] = w.Version
	payload["method"] = w.Method
	payload["ts"] = w.Timestamp
	payload["response"] = w.Response

	signer := utils.NewWsDataSigner(payload, token, utils.AlgoSHA256)
	return signer.Verify(w.Sign)
}
func (w *MCPSdkResponse) McpResponse() (mcp.ServerResult, error) {
	if w.Response == "" {
		return "", errors.New("response is nil")
	}
	return w.Response, nil
}
