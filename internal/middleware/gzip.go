package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

var compressibleTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      io.Writer
	wroteHeader bool
	compress    bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		ct := w.Header().Get("Content-Type")
		if ct == "" {
			ct = http.DetectContentType(b)
			w.Header().Set("Content-Type", ct)
		}
		w.chooseCompression(ct)
		w.wroteHeader = true
	}
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	ct := w.Header().Get("Content-Type")
	w.chooseCompression(ct)
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) chooseCompression(ct string) {
	if w.compress {
		return
	}
	base := ct
	if i := strings.Index(ct, ";"); i >= 0 {
		base = ct[:i]
	}
	base = strings.TrimSpace(base)

	if compressibleTypes[base] {
		gz, err := gzip.NewWriterLevel(w.ResponseWriter, gzip.BestSpeed)
		if err != nil {
			return
		}
		w.compress = true
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.writer = gz
	}
}

func (w *gzipResponseWriter) Close() error {
	if gz, ok := w.writer.(*gzip.Writer); ok {
		return gz.Close()
	}
	return nil
}

// GzipMiddleware распаковывает gzip-тела запросов и сжимает подходящие ответы,
// если клиент поддерживает gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "failed to decompress request", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
			r.Header.Del("Content-Encoding")
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gw := &gzipResponseWriter{
			ResponseWriter: w,
			writer:         w,
		}
		defer gw.Close()

		next.ServeHTTP(gw, r)
	})
}
