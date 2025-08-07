package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	AlgoSHA256 AlgoKind = "HMAC-SHA256"
)

var (
	signerMap = map[AlgoKind]IAlgo{
		AlgoSHA256: &Sha256Algo{},
	}
)

type AlgoKind string

type IAlgo interface {
	Kind() string
	Sign(data []byte, salt string) (string, error)
	Verify(data []byte, salt string, sign string) (bool, error)
}

type Sha256Algo struct {
}

func (s *Sha256Algo) Kind() string {
	return string(AlgoSHA256)
}

func (s *Sha256Algo) Sign(data []byte, salt string) (string, error) {
	sign := hmac.New(sha256.New, []byte(salt))
	sign.Write(data)
	_sign := hex.EncodeToString(sign.Sum(nil))
	fmt.Println("---------------- Debug Sign Start ----------------")
	fmt.Println("salt:", salt)
	fmt.Println("data:", string(data))
	return strings.ToUpper(_sign), nil
}

func (s *Sha256Algo) Verify(data []byte, salt string, sign string) (bool, error) {
	signer := hmac.New(sha256.New, []byte(salt))
	signer.Write(data)
	_sign := hex.EncodeToString(signer.Sum(nil))
	// TODO: Debug Logger printer
	fmt.Println("------------ Debug Verify Sign Start -------------")
	fmt.Println("salt:", salt)
	fmt.Println("data:", string(data))
	fmt.Println("params_sign:", sign)
	fmt.Println("calcul_sign:", _sign)

	return strings.ToUpper(_sign) == sign, nil
}
