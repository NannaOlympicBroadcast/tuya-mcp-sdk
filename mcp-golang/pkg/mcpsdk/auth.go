package mcpsdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"mcp-sdk/pkg/utils"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuthToken struct {
	endpoint     string
	accessKey    string
	accessSecret string
	authResponse
}

type authData struct {
	Token    string `json:"token"`
	ClientId string `json:"client_id"`
}

type authResponse struct {
	Data    authData `json:"data"`
	Success bool     `json:"success"`
}

func NewAuthToken(endpoint, accessKey, accessSecret string) *AuthToken {
	return &AuthToken{
		endpoint:     endpoint,
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
}

func (a *AuthToken) Auth() error {
	header := map[string]string{}
	header["access_id"] = a.accessKey
	header["t"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	header["nonce"] = strings.ReplaceAll(uuid.New().String(), "-", "")[:32]
	header["sign_method"] = "HMAC-SHA256"

	u, err := a.url(UrlTypeAuth)
	if err != nil {
		return err
	}

	authUrl, err := url.Parse(u)
	if err != nil {
		return err
	}

	signer := utils.NewRestfulSigner(utils.AlgoSHA256, a.accessSecret, utils.WithSignerHeader(header), utils.WithSignerPath(authUrl.Path))
	sign, err := signer.Sign()
	if err != nil {
		return err
	}
	header["sign"] = sign

	println("request auth api to url:", authUrl.String())

	resp, err := utils.HttpGet(authUrl.String(), header)
	if err != nil {
		return err
	}

	println("auth response: ", string(resp))

	if err = json.Unmarshal([]byte(resp), &a.authResponse); err != nil {
		return err
	}

	if !a.authResponse.Success {
		return fmt.Errorf("auth response token is empty, err: %s", string(resp))
	}

	return nil
}

func (a *AuthToken) ConnectHeader() (urlAddr string, header map[string]string, err error) {
	header = map[string]string{}
	header["access_id"] = a.accessKey
	header["t"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	header["nonce"] = strings.ReplaceAll(uuid.New().String(), "-", "")[:32]
	header["sign_method"] = "HMAC-SHA256"

	urlAddr = a.connectUrl(a.authResponse.Data.ClientId)
	println("connect websocket to url: ", urlAddr)
	urlPath, err := url.Parse(urlAddr)
	if err != nil {
		return "", nil, err
	}

	query := urlPath.Query()

	signer := utils.NewRestfulSigner(utils.AlgoSHA256, a.authResponse.Data.Token, utils.WithSignerHeader(header), utils.WithSignerQuery(query), utils.WithSignerPath(urlPath.Path))
	sign, err := signer.Sign()
	if err != nil {
		return "", nil, err
	}
	header["sign"] = sign
	return urlAddr, header, nil
}

func (a *AuthToken) connectUrl(clientId string) string {
	url, err := a.url(UrlTypeConnect)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s?client_id=%s", url, clientId)
}

func (a *AuthToken) authUrl() string {
	url, err := a.url(UrlTypeAuth)
	if err != nil {
		return ""
	}
	return url
}

func (a *AuthToken) url(urlType urlType) (string, error) {
	u, err := url.Parse(a.endpoint)
	if err != nil {
		return "", err
	}
	switch urlType {
	case UrlTypeAuth:
		u.Path = "/v1/client/registration"
	case UrlTypeConnect:
		switch u.Scheme {
		case "http":
			u.Scheme = "ws"
		case "https":
			u.Scheme = "wss"
		}
		u.Path = "/ws/mcp"
	default:
		return "", errors.New("not supported url connection type")
	}
	return u.String(), nil
}

type urlType string

const (
	UrlTypeAuth    urlType = "auth"
	UrlTypeConnect urlType = "connect"
)
