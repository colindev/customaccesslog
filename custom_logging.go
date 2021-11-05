package handlers

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gorilla/handlers"
)

const lowerhex = "0123456789abcdef"

type ctxKey int
type CtxErr struct {
	Err error
}

func (e *CtxErr) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

var (
	errKey    ctxKey = 1
	targetKey ctxKey = 2
)

func PrepareCustomLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		h.ServeHTTP(w, r.WithContext(context.WithValue(ctx, errKey, &CtxErr{nil})))
	})
}

func RecordBackend(r *http.Request, be *url.URL) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, targetKey, be.String())
	newReq := r.WithContext(ctx)
	*r = *newReq
	return newReq
}

func RequestCtxWithError(r *http.Request, err error) {
	v := r.Context().Value(errKey)
	eCtx, ok := v.(*CtxErr)
	if ok {
		eCtx.Err = err
	}
}

// implement gorilla/handlers/LogFormatter
func WriteCustomLog(writer io.Writer, params handlers.LogFormatterParams) {
	buf := buildCommonLogLine(params.Request, params.URL, params.TimeStamp, params.StatusCode, params.Size)
	buf = append(buf, fmt.Sprintf(" XFF(%s)", params.Request.Header.Get("X-Forwarded-For"))...)
	buf = append(buf, " host("...)
	buf = append(buf, params.Request.Host...)
	buf = append(buf, ")"...)
	buf = append(buf, " upstream("...)
	buf = append(buf, time.Now().Sub(params.TimeStamp).String()...)
	buf = append(buf, ")"...)
	if ctxValue := params.Request.Context().Value(errKey); ctxValue != nil {
		if e, ok := ctxValue.(*CtxErr); ok && e.Err != nil {
			buf = append(buf, " err: "...)
			buf = append(buf, e.Error()...)
		}
	}
	buf = append(buf, '\n')
	writer.Write(buf)
}

func buildCommonLogLine(req *http.Request, url url.URL, ts time.Time, status int, size int) []byte {
	username := "-"
	if url.User != nil {
		if name := url.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		host = req.RemoteAddr
	}

	uri := req.RequestURI
	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if req.ProtoMajor == 2 && req.Method == "CONNECT" {
		uri = req.Host
	}
	if uri == "" {
		uri = url.RequestURI()
	}

	ctx := req.Context()
	if backend := ctx.Value(targetKey); backend != nil {
		uri = uri + " --> " + backend.(string)
	}

	buf := make([]byte, 0, 3*(len(host)+len(username)+len(req.Method)+len(uri)+len(req.Proto)+50)/2)
	buf = append(buf, host...)
	buf = append(buf, " - "...)
	buf = append(buf, username...)
	buf = append(buf, " ["...)
	buf = append(buf, ts.Format("02/Jan/2006:15:04:05 -0700")...)
	buf = append(buf, `] "`...)
	buf = append(buf, req.Method...)
	buf = append(buf, " "...)
	buf = appendQuoted(buf, uri)
	buf = append(buf, " "...)
	buf = append(buf, req.Proto...)
	buf = append(buf, `" `...)
	buf = append(buf, strconv.Itoa(status)...)
	buf = append(buf, " "...)
	buf = append(buf, strconv.Itoa(size)...)
	return buf
}

func appendQuoted(buf []byte, s string) []byte {
	var runeTmp [utf8.UTFMax]byte
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		if r == rune('"') || r == '\\' { // always backslashed
			buf = append(buf, '\\')
			buf = append(buf, byte(r))
			continue
		}
		if strconv.IsPrint(r) {
			n := utf8.EncodeRune(runeTmp[:], r)
			buf = append(buf, runeTmp[:n]...)
			continue
		}
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[0]>>4])
				buf = append(buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	return buf

}
