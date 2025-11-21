package csvutil

import (
	"encoding/csv"
	"io"
)

// Writer handles CSV writing.
type Writer struct {
	writer *csv.Writer
}

// NewWriter creates a new CSV writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: csv.NewWriter(w),
	}
}

// WriteHeader writes the CSV header row.
func (w *Writer) WriteHeader(headers []string) error {
	return w.writer.Write(headers)
}

// WriteRow writes a single CSV row.
func (w *Writer) WriteRow(row []string) error {
	return w.writer.Write(row)
}

// WriteAll writes multiple rows at once.
func (w *Writer) WriteAll(rows [][]string) error {
	return w.writer.WriteAll(rows)
}

// Flush flushes any buffered data to the underlying writer.
func (w *Writer) Flush() error {
	w.writer.Flush()
	return w.writer.Error()
}

