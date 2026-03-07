package exchangerate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// FileProvider reads exchange rates from a local JSON file.
// Expected format:
//
//	{"base": "USD", "rates": {"EUR": 0.92, "GBP": 0.79, ...}}
//
// The base field in the file must match the base argument passed to FetchRates,
// otherwise an error is returned.
type FileProvider struct {
	path string
}

func NewFileProvider(path string) *FileProvider {
	return &FileProvider{path: path}
}

type fileRates struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

func (p *FileProvider) FetchRates(ctx context.Context, base string) (map[string]float64, error) {
	f, err := os.Open(p.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data fileRates
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	if data.Base != base {
		return nil, fmt.Errorf("file provider: base currency mismatch: file has %q, requested %q", data.Base, base)
	}
	data.Rates[base] = 1.0
	return data.Rates, nil
}
