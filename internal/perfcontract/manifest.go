package perfcontract

import (
	"encoding/json"
	"fmt"
	"os"
)

type BenchEntry struct {
	Name            string  `json:"name"`
	Path            string  `json:"path"`
	LLVMRatioTarget float64 `json:"llvm_ratio_target"`
}

type BenchManifest struct {
	BaselineThresholdPercent int          `json:"baseline_threshold_percent"`
	Benchmarks               []BenchEntry `json:"benchmarks"`
}

func LoadBenchManifest(path string) (*BenchManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read benchmark manifest: %w", err)
	}
	var manifest BenchManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode benchmark manifest: %w", err)
	}
	return &manifest, nil
}
