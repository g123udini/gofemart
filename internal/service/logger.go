package service

import (
	"go.uber.org/zap"
	"net/http"
)

func NewLogger() *zap.SugaredLogger {
	core, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	return core.Sugar()
}

type (
	responseData struct {
		Status int
		Size   int
	}

	LoggingResponseWriter struct {
		http.ResponseWriter
		ResponseData *responseData
	}
)

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{ResponseWriter: w, ResponseData: &responseData{}}
}

func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.ResponseData.Status = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.ResponseData.Size += size
	return size, err
}
