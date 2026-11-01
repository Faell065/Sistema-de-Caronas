package servidor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ClienteREST struct {
	Cliente   *http.Client
	EmpresaID string
	ChaveHMAC string
}

func NovoClienteREST(empresaID, chaveHMAC string, timeout time.Duration) *ClienteREST {
	return &ClienteREST{
		Cliente:   &http.Client{Timeout: timeout},
		EmpresaID: empresaID,
		ChaveHMAC: chaveHMAC,
	}
}

func (c *ClienteREST) Chamar(methodo, url, idempotencyKey string, corpo interface{}) (int, []byte, error) {
	var payload []byte
	var err error
	if corpo != nil {
		payload, err = json.Marshal(corpo)
		if err != nil {
			return 0, nil, err
		}
	}

	request, err := http.NewRequest(methodo, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-VaiJunto-Empresa-ID", c.EmpresaID)
	request.Header.Set("X-VaiJunto-Idempotency-Key", idempotencyKey)
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	request.Header.Set("X-VaiJunto-Timestamp", timestamp)
	if c.ChaveHMAC != "" {
		assinatura := AssinarCorpo(c.ChaveHMAC, methodo, request.URL.RequestURI(), timestamp, c.EmpresaID, idempotencyKey, string(payload))
		request.Header.Set("X-VaiJunto-Signature", assinatura)
	}

	response, err := c.Cliente.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	resposta, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return response.StatusCode, resposta, err
}

func (c *ClienteREST) ChamarComRepeticao(methodo, url, idempotencyKey string, corpo interface{}, tentativas int) (int, []byte, error) {
	if tentativas < 1 {
		tentativas = 1
	}
	var ultimoErro error
	for tentativa := 0; tentativa < tentativas; tentativa++ {
		status, resposta, err := c.Chamar(methodo, url, idempotencyKey, corpo)
		if err == nil {
			if status >= 500 || status == http.StatusGatewayTimeout {
				ultimoErro = fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(resposta)))
			} else {
				return status, resposta, nil
			}
		} else {
			ultimoErro = err
		}
		if tentativa+1 < tentativas {
			time.Sleep(time.Duration(100*(tentativa+1)) * time.Millisecond)
		}
	}
	return 0, nil, ultimoErro
}
