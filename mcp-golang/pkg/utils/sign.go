package utils

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Signer interface {
	Sign() (string, error)
	Verify(sign string) (bool, error)
}

// HTTP 通用鉴权
type RestfulSigner struct {
	params          map[string][]string
	path            string
	header          map[string][]string
	payload         []byte
	salt            string
	signerAlgorithm IAlgo
}

type RestfulSignerOption func(*RestfulSigner)

func WithSignerType(signerType AlgoKind) RestfulSignerOption {
	return func(s *RestfulSigner) {
		s.signerAlgorithm = signerMap[signerType]
	}
}

func WithSignerSalt(salt string) RestfulSignerOption {
	return func(s *RestfulSigner) {
		s.salt = salt
	}
}

func WithSignerQuery(query url.Values) RestfulSignerOption {
	return func(s *RestfulSigner) {
		s.params = query
	}
}

func WithSignerPath(path string) RestfulSignerOption {
	return func(s *RestfulSigner) {
		s.path = path
	}
}

func WithSignerHeader(header map[string]string) RestfulSignerOption {
	headerMap := map[string][]string{}
	for key, value := range header {
		headerMap[key] = []string{value}
	}
	return func(s *RestfulSigner) {
		s.header = headerMap
	}
}

func WithSignerPayload(payload []byte) RestfulSignerOption {
	return func(s *RestfulSigner) {
		s.payload = payload
	}
}

func NewRestfulSigner(signerType AlgoKind, salt string, options ...RestfulSignerOption) Signer {
	signer := &RestfulSigner{
		signerAlgorithm: signerMap[signerType],
		salt:            salt,
	}

	for _, option := range options {
		option(signer)
	}
	return signer
}

func (s *RestfulSigner) headerStr() string {
	if s.header == nil {
		return ""
	}

	header := make(map[string]string)
	for key, values := range s.header {
		header[strings.ToLower(key)] = strings.Join(values, ",")
	}

	clientId := header["access_id"]
	timestamp := header["t"]
	signMethod := header["sign_method"]
	nonce := header["nonce"]

	headerParamsStr := fmt.Sprintf("%s\n%s\n%s\n%s\n", clientId, timestamp, signMethod, nonce)

	signatureHeaders := strings.Join(s.header["signature_headers"], ",")
	if signatureHeaders != "" {
		signatureHeaders := strings.Split(signatureHeaders, ",")
		for _, headerKey := range signatureHeaders {
			headerValue := strings.Join(s.header[strings.TrimSpace(headerKey)], ",")
			headerParamsStr += fmt.Sprintf("%s:%s\n", strings.TrimSpace(headerKey), strings.TrimSpace(headerValue))
		}
	}
	return headerParamsStr
}

func (s *RestfulSigner) queryParamsStr() string {
	if len(s.params) == 0 {
		return ""
	}
	params := map[string]string{}
	for key, values := range s.params {
		params[key] = strings.Join(values, ",")
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	paramsStr := ""
	for _, key := range keys {
		paramsStr += fmt.Sprintf("%s=%s&", key, params[key])
	}
	return paramsStr[:len(paramsStr)-1]
}

func (s *RestfulSigner) payloadStr() string {
	if len(s.payload) == 0 {
		return ""
	}
	return string(s.payload)
}

func (s *RestfulSigner) url() string {
	if s.path == "" {
		return ""
	}
	return s.path
}

func (s *RestfulSigner) genSignStr() string {
	return s.headerStr() + "\n" + s.queryParamsStr() + "\n" + s.payloadStr() + "\n" + s.url()
}

func (s *RestfulSigner) Sign() (string, error) {
	sign, err := s.signerAlgorithm.Sign([]byte(s.genSignStr()), s.salt)
	if err != nil {
		return "", err
	}
	return sign, nil
}

func (s *RestfulSigner) Verify(sign string) (bool, error) {
	return s.signerAlgorithm.Verify([]byte(s.genSignStr()), s.salt, sign)
}

// Websocket 数据鉴权
type WsDataSigner struct {
	payload         map[string]string
	salt            string
	signerAlgorithm IAlgo
}

func NewWsDataSigner(payload map[string]string, salt string, signerType AlgoKind) *WsDataSigner {
	return &WsDataSigner{
		payload:         payload,
		salt:            salt,
		signerAlgorithm: signerMap[signerType],
	}
}

func (s *WsDataSigner) genSignStr() string {
	signStr := ""
	keys := make([]string, 0, len(s.payload))
	for key := range s.payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "sign" {
			continue
		}
		signStr += fmt.Sprintf("%s:%s\n", key, s.payload[key])
	}
	return signStr[:len(signStr)-1]
}

func (s *WsDataSigner) Sign() (string, error) {
	sign, err := s.signerAlgorithm.Sign([]byte(s.genSignStr()), s.salt)
	if err != nil {
		return "", err
	}
	return sign, nil
}

func (s *WsDataSigner) Verify(sign string) (bool, error) {
	return s.signerAlgorithm.Verify([]byte(s.genSignStr()), s.salt, sign)
}
