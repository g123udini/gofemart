package service

import (
	"database/sql"
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"
)

type memDTO struct {
	Gauges   map[string]float64 `json:"gauges"`
	Counters map[string]int64   `json:"counters"`
}

func TestNewMemStorage(t *testing.T) {
	db := &sql.DB{} // просто не-nil заглушка

	ms := NewMemStorage("/tmp/test.json", db)

	if ms == nil {
		t.Fatalf("NewMemStorage() returned nil")
	}
	if ms.filepath != "/tmp/test.json" {
		t.Errorf("filepath = %q, want %q", ms.filepath, "/tmp/test.json")
	}
	if ms.db != db {
		t.Errorf("db not set correctly")
	}
	if ms.maxRetry != 3 {
		t.Errorf("maxRetry = %d, want 3", ms.maxRetry)
	}
	if ms.gauges == nil {
		t.Errorf("gauges map is nil")
	}
	if ms.counters == nil {
		t.Errorf("counters map is nil")
	}
}

func TestMemStorage_GetCounter(t *testing.T) {
	ms := &MemStorage{
		counters: map[string]int64{
			"hits":   10,
			"errors": 2,
		},
	}

	tests := []struct {
		name   string
		key    string
		want   int64
		wantOk bool
	}{
		{
			name:   "existing key",
			key:    "hits",
			want:   10,
			wantOk: true,
		},
		{
			name:   "missing key",
			key:    "miss",
			want:   0,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ms.GetCounter(tt.key)
			if got != tt.want {
				t.Errorf("GetCounter(%q) got = %v, want %v", tt.key, got, tt.want)
			}
			if ok != tt.wantOk {
				t.Errorf("GetCounter(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
		})
	}
}

func TestMemStorage_GetGauge(t *testing.T) {
	ms := &MemStorage{
		gauges: map[string]float64{
			"load":  0.75,
			"usage": 0.5,
		},
	}

	tests := []struct {
		name   string
		key    string
		want   float64
		wantOk bool
	}{
		{
			name:   "existing key",
			key:    "load",
			want:   0.75,
			wantOk: true,
		},
		{
			name:   "missing key",
			key:    "miss",
			want:   0,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ms.GetGauge(tt.key)
			if got != tt.want {
				t.Errorf("GetGauge(%q) got = %v, want %v", tt.key, got, tt.want)
			}
			if ok != tt.wantOk {
				t.Errorf("GetGauge(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
		})
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	ms := &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	err := ms.UpdateGauge("temp", 36.6)
	if err != nil {
		t.Fatalf("UpdateGauge() unexpected error: %v", err)
	}

	if val, ok := ms.GetGauge("temp"); !ok || val != 36.6 {
		t.Fatalf("gauge not updated correctly, got (%v,%v), want (36.6,true)", val, ok)
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	ms := &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	if err := ms.UpdateCounter("hits", 5); err != nil {
		t.Fatalf("UpdateCounter() unexpected error: %v", err)
	}
	if err := ms.UpdateCounter("hits", 3); err != nil {
		t.Fatalf("UpdateCounter() unexpected error: %v", err)
	}

	val, ok := ms.GetCounter("hits")
	if !ok {
		t.Fatalf("counter 'hits' not found")
	}
	if val != 8 {
		t.Fatalf("counter value = %d, want 8", val)
	}
}

func TestMemStorage_MarshalJSON(t *testing.T) {
	ms := &MemStorage{
		gauges: map[string]float64{
			"load": 0.9,
		},
		counters: map[string]int64{
			"hits": 42,
		},
	}

	data, err := ms.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error: %v", err)
	}

	var dto memDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		t.Fatalf("unmarshal back failed: %v", err)
	}

	want := memDTO{
		Gauges:   map[string]float64{"load": 0.9},
		Counters: map[string]int64{"hits": 42},
	}

	if !reflect.DeepEqual(dto, want) {
		t.Errorf("MarshalJSON() dto = %+v, want %+v", dto, want)
	}
}

func TestMemStorage_UnmarshalJSON(t *testing.T) {
	jsonData := []byte(`{
		"gauges": {"load": 0.5, "usage": 0.75},
		"counters": {"hits": 10, "errors": 1}
	}`)

	ms := &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	if err := ms.UnmarshalJSON(jsonData); err != nil {
		t.Fatalf("UnmarshalJSON() error: %v", err)
	}

	wantGauges := map[string]float64{"load": 0.5, "usage": 0.75}
	wantCounters := map[string]int64{"hits": 10, "errors": 1}

	if !reflect.DeepEqual(ms.gauges, wantGauges) {
		t.Errorf("gauges = %+v, want %+v", ms.gauges, wantGauges)
	}
	if !reflect.DeepEqual(ms.counters, wantCounters) {
		t.Errorf("counters = %+v, want %+v", ms.counters, wantCounters)
	}
}

func TestMemStorage_FlushToFile_and_ReadFromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "memstorage-*.json")
	if err != nil {
		t.Fatalf("CreateTemp error: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	ms := &MemStorage{
		gauges: map[string]float64{
			"load":  0.7,
			"usage": 0.3,
		},
		counters: map[string]int64{
			"hits":   100,
			"errors": 2,
		},
		filepath: tmpFile.Name(),
	}

	// записали в файл
	if err := ms.FlushToFile(); err != nil {
		t.Fatalf("FlushToFile() error: %v", err)
	}

	ms2 := &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}

	if err := ms2.ReadFromFile(tmpFile.Name()); err != nil {
		t.Fatalf("ReadFromFile() error: %v", err)
	}

	if !reflect.DeepEqual(ms.gauges, ms2.gauges) {
		t.Errorf("gauges after ReadFromFile = %+v, want %+v", ms2.gauges, ms.gauges)
	}
	if !reflect.DeepEqual(ms.counters, ms2.counters) {
		t.Errorf("counters after ReadFromFile = %+v, want %+v", ms2.counters, ms.counters)
	}
}

func TestMemStorage_saveDB_NoDB(t *testing.T) {
	ms := &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		mu:       sync.RWMutex{},
		filepath: "",
		db:       nil, // главное — nil, чтобы saveDB просто вернул nil
		maxRetry: 3,
	}

	if err := ms.saveDB("metric", "42"); err != nil {
		t.Fatalf("saveDB() with nil db returned error: %v", err)
	}
}
