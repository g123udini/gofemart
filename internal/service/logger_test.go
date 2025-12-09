package service

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestLoggingResponseWriter_Write(t *testing.T) {
	type fields struct {
		ResponseWriter http.ResponseWriter
		ResponseData   *responseData
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     int
		wantErr  bool
		wantSize int
		wantBody string
	}{
		{
			name: "write once",
			fields: fields{
				ResponseWriter: httptest.NewRecorder(),
				ResponseData:   &responseData{},
			},
			args: args{
				b: []byte("hello"),
			},
			want:     len("hello"),
			wantErr:  false,
			wantSize: len("hello"),
			wantBody: "hello",
		},
		{
			name: "write twice accumulates size",
			fields: fields{
				ResponseWriter: httptest.NewRecorder(),
				ResponseData:   &responseData{},
			},
			args: args{
				b: []byte(" world"),
			},
			want:     len(" world"),
			wantErr:  false,
			wantSize: len("hello world"),
			wantBody: "hello world",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			lrw := &LoggingResponseWriter{
				ResponseWriter: tt.fields.ResponseWriter,
				ResponseData:   tt.fields.ResponseData,
			}

			rr := tt.fields.ResponseWriter.(*httptest.ResponseRecorder)

			if tt.name == "write twice accumulates size" {
				if _, err := lrw.Write([]byte("hello")); err != nil {
					t.Fatalf("pre write error: %v", err)
				}
			}

			got, err := lrw.Write(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Write() got = %v, want %v", got, tt.want)
			}
			if lrw.ResponseData.Size != tt.wantSize {
				t.Errorf("ResponseData.Size = %v, want %v", lrw.ResponseData.Size, tt.wantSize)
			}
			if body := rr.Body.String(); body != tt.wantBody {
				t.Errorf("underlying body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	type fields struct {
		ResponseWriter http.ResponseWriter
		ResponseData   *responseData
	}
	type args struct {
		code int
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantStatus   int
		wantHTTPCode int
	}{
		{
			name: "set status",
			fields: fields{
				ResponseWriter: httptest.NewRecorder(),
				ResponseData:   &responseData{},
			},
			args: args{
				code: http.StatusTeapot,
			},
			wantStatus:   http.StatusTeapot,
			wantHTTPCode: http.StatusTeapot,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			lrw := &LoggingResponseWriter{
				ResponseWriter: tt.fields.ResponseWriter,
				ResponseData:   tt.fields.ResponseData,
			}
			lrw.WriteHeader(tt.args.code)

			if lrw.ResponseData.Status != tt.wantStatus {
				t.Errorf("ResponseData.Status = %v, want %v", lrw.ResponseData.Status, tt.wantStatus)
			}

			rr := tt.fields.ResponseWriter.(*httptest.ResponseRecorder)
			if rr.Code != tt.wantHTTPCode {
				t.Errorf("underlying status code = %v, want %v", rr.Code, tt.wantHTTPCode)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	t.Run("returns non-nil logger", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatalf("NewLogger() returned nil")
		}

		logger.Infof("test log: %s", "ok")
	})
}

func TestNewLoggingResponseWriter(t *testing.T) {
	t.Run("wraps writer and inits responseData", func(t *testing.T) {
		rr := httptest.NewRecorder()

		lrw := NewLoggingResponseWriter(rr)

		if lrw.ResponseWriter != rr {
			t.Errorf("ResponseWriter not set correctly")
		}
		if lrw.ResponseData == nil {
			t.Fatalf("ResponseData is nil")
		}
		if !reflect.DeepEqual(*lrw.ResponseData, responseData{}) {
			t.Errorf("ResponseData not zero-initialized, got %+v", lrw.ResponseData)
		}
	})
}
