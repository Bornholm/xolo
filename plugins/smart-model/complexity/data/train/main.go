package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bornholm/xolo/plugins/smart-model/complexity"
)

func main() {
	trainOut := flag.String("train", "", "Train model and save to this path, then exit")
	csvPath := flag.String("csv", "", "Path to a CSV training file (columns: label,text) — used with -train")
	flag.Parse()

	// Train-and-save mode
	nb := complexity.NewNaiveBayes(1.0, 2)

	var examples []complexity.TrainingExample
	if *csvPath == "" {
		fmt.Fprintf(os.Stderr, "csv flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	var err error
	examples, err = loadCSV(*csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading CSV: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d examples from %s\n", len(examples), *csvPath)

	nb.Train(examples)
	if err := nb.SaveModel(*trainOut); err != nil {
		fmt.Fprintf(os.Stderr, "error saving model: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Model saved to %s\n", *trainOut)
	fmt.Println(nb.Summary())
}

// loadCSV reads a CSV file with columns "label" and "text" (header required).
// The column order is detected from the header row.
func loadCSV(path string) ([]complexity.TrainingExample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	labelIdx, textIdx := -1, -1
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "label":
			labelIdx = i
		case "text":
			textIdx = i
		}
	}
	if labelIdx < 0 || textIdx < 0 {
		return nil, fmt.Errorf("CSV must have 'label' and 'text' columns, got: %v", header)
	}

	var examples []complexity.TrainingExample
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		if len(row) <= labelIdx || len(row) <= textIdx {
			continue
		}
		label := strings.TrimSpace(row[labelIdx])
		text := strings.TrimSpace(row[textIdx])
		if label == "" || text == "" {
			continue
		}
		examples = append(examples, complexity.TrainingExample{Label: label, Text: text})
	}

	if len(examples) == 0 {
		return nil, fmt.Errorf("no valid examples found in %s", path)
	}
	return examples, nil
}
