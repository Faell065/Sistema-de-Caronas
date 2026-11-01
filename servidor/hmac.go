package servidor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func AssinarCorpo(chave, metodo, caminho, timestamp, empresaID, idempotencyKey, corpo string) string {
	base := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", metodo, caminho, timestamp, empresaID, idempotencyKey, corpo)
	mac := hmac.New(sha256.New, []byte(chave))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

func ValidarAssinatura(chave string, r *http.Request, corpo []byte) bool {
	if chave == "" {
		return true
	}
	timestamp := r.Header.Get("X-VaiJunto-Timestamp")
	nanos, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || timestampVencido(nanos, time.Now()) {
		return false
	}
	signatura := r.Header.Get("X-VaiJunto-Signature")
	esperada := AssinarCorpo(chave, r.Method, r.URL.RequestURI(), timestamp,
		r.Header.Get("X-VaiJunto-Empresa-ID"), r.Header.Get("X-VaiJunto-Idempotency-Key"), string(corpo))
	return hmac.Equal([]byte(signatura), []byte(esperada))
}

func timestampVencido(timestamp int64, agora time.Time) bool {
	diferenca := agora.Unix() - timestamp/1e9
	return diferenca > 300 || diferenca < -300
}
