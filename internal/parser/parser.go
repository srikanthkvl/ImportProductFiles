package parser

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

// Record is a generic parsed map that will be validated per product schema.
type Record map[string]string

// Detect and parse file types: CSV, TSV, simple text (key=value per line), and basic Excel (xlsx) placeholder.

// Parse reads the content and returns a slice of records keyed by header names.
func Parse(filename string, r io.Reader) ([]Record, error) {
	ct := contentTypeFromExt(filename)
	switch ct {
	case "text/csv":
		return parseCSV(r, ',')
	case "text/tab-separated-values":
		return parseCSV(r, '\t')
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return parseXLSX(r)
	default:
		return parseKV(r)
	}
}

func contentTypeFromExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".csv" {
		return "text/csv"
	}
	if ext == ".tsv" || ext == ".tab" {
		return "text/tab-separated-values"
	}
	if ext == ".xlsx" {
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	// try system
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "text/plain"
}

func parseCSV(r io.Reader, sep rune) ([]Record, error) {
	cr := csv.NewReader(r)
	cr.Comma = sep
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	var out []Record
	for i := 1; i < len(records); i++ {
		row := records[i]
		rec := Record{}
		for j := 0; j < len(headers) && j < len(row); j++ {
			rec[strings.TrimSpace(headers[j])] = strings.TrimSpace(row[j])
		}
		out = append(out, rec)
	}
	return out, nil
}

// parseKV supports lines like key=value; blank lines and lines starting with # are ignored.
func parseKV(r io.Reader) ([]Record, error) {
	s := bufio.NewScanner(r)
	res := []Record{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid kv line: %s", line)
		}
		res = append(res, Record{"key": strings.TrimSpace(parts[0]), "value": strings.TrimSpace(parts[1])})
	}
	return res, s.Err()
}

// parseXLSX is a placeholder that returns an error explaining xlsx requires full file. We will buffer and parse with a library.
func parseXLSX(r io.Reader) ([]Record, error) {
	return nil, errors.New("xlsx parsing requires full file; provide .csv/.tsv for now")
}


