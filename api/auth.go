package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func nonce() string {
	nonce := make([]byte, 16)
	n := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(nonce, uint64(n))
	binary.LittleEndian.PutUint64(nonce[8:], rand.Uint64())
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(nonce[:4]),
		hex.EncodeToString(nonce[4:6]),
		hex.EncodeToString(nonce[6:8]),
		hex.EncodeToString(nonce[8:10]),
		hex.EncodeToString(nonce[10:]),
	)
}

func sign(key, secret, method, host, path, query, contentType, nonce, timestamp, params string) string {
	h := hmac.New(sha256.New, []byte(secret))
	fmt.Fprint(
		h,
		"BITSTAMP ", key,
		method, host, path, query,
		contentType,
		nonce, timestamp,
		"v2",
		params,
	)

	return hex.EncodeToString(h.Sum(nil))
}

func Sign(apiKey, apiSecret string, r *http.Request) error {
	var bodyData []byte
	if r.GetBody != nil {
		body, err := r.GetBody()
		if err != nil {
			panic(err)
		}
		bodyData, err = io.ReadAll(body)
		if err != nil {
			panic(err)
		}
	}

	nowStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	nonce := nonce()
	sig := sign(
		apiKey,
		apiSecret,
		r.Method,
		r.URL.Hostname(),
		r.URL.Path,
		r.URL.Query().Encode(),
		r.Header.Get("Content-Type"),
		nonce,
		nowStr,
		string(bodyData),
	)

	r.Header.Set("X-Auth", fmt.Sprintf("BITSTAMP %s", apiKey))
	r.Header.Set("X-Auth-Signature", sig)
	r.Header.Set("X-Auth-Nonce", nonce)
	r.Header.Set("X-Auth-Version", "v2")
	r.Header.Set("X-Auth-Timestamp", nowStr)

	return nil
}
