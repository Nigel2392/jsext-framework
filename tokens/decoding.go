//go:build js && wasm
// +build js,wasm

package tokens

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/Nigel2392/jsext/requester/decoder"
)

type JWTToken struct {
	Header    Header    `json:"header"`
	Payload   Payload   `json:"payload"`
	Signature Signature `json:"signature"`
}

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type Payload map[string]interface{}

func (p Payload) Get(key string) interface{} {
	return p[key]
}

func (p Payload) GetTime(key string) time.Time {
	var t = p.Get(key)
	if t == nil {
		return time.Time{}
	}
	var tFloat, ok = t.(float64)
	if !ok {
		return time.Time{}
	}
	return time.Unix(int64(tFloat), 0)
}

type Signature string

func DecodeToken(token string) (JWTToken, error) {
	var parts = strings.Split(token, ".")
	if len(parts) != 3 {
		return JWTToken{}, errors.New("invalid token")
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return JWTToken{}, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return JWTToken{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return JWTToken{}, err
	}
	var t = JWTToken{
		Header:    Header{},
		Payload:   Payload{},
		Signature: Signature(signature),
	}

	var m = decoder.ParseBytes(header).Value().(map[string]interface{})
	t.Header.Alg = m["alg"].(string)
	t.Header.Typ = m["typ"].(string)
	t.Payload = decoder.ParseBytes(payload).Value().(map[string]interface{})
	return t, nil
}
