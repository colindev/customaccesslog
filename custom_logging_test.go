package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	gh "github.com/gorilla/handlers"
)

func Test_CustomLog(t *testing.T) {

	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RequestCtxWithError(r, errors.New("test-error"))
		time.Sleep(time.Millisecond * 30)
		w.Write([]byte(strings.Repeat("x", 2000)))
	})

	buf := bytes.NewBuffer(nil)
	h = gh.CustomLoggingHandler(buf, h, WriteCustomLog)
	h = PrepareCustomLog(h)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "https://1.2.3.4:8000", nil)
	r.Header.Set("X-Forwarded-For", "255.255.255.255, 244.244.244.244, 233.233.233.233, 222.222.222.222, 211.211.211.211, 200.200.200.200")
	be, _ := url.Parse("https://echo.colindev.io/api")
	RecordBackend(r, be)

	h.ServeHTTP(w, r)

	t.Log(buf.Len())
	t.Log(buf.String())
}
