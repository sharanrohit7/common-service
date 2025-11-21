package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// RowValidator validates a CSV row.
type RowValidator func(row []string, rowNum int) error

// ParserConfig configures CSV parsing behavior.
type ParserConfig struct {
	HasHeader      bool
	Comma          rune
	Comment        rune
	LazyQuotes     bool
	TrimLeadingSpace bool
	SkipEmptyRows  bool
	Validators     []RowValidator
}

// DefaultParserConfig returns a default parser configuration.
func DefaultParserConfig() ParserConfig {
	return ParserConfig{
		HasHeader:      true,
		Comma:          ',',
		LazyQuotes:     false,
		TrimLeadingSpace: true,
		SkipEmptyRows:  true,
		Validators:     []RowValidator{},
	}
}

// Parser handles CSV parsing with validation.
type Parser struct {
	config ParserConfig
	header []string
}

// NewParser creates a new CSV parser.
func NewParser(config ParserConfig) *Parser {
	return &Parser{
		config: config,
	}
}

// Parse parses CSV data from a reader and calls the handler for each row.
func (p *Parser) Parse(reader io.Reader, handler func(rowNum int, headers []string, row []string) error) error {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = p.config.Comma
	csvReader.Comment = p.config.Comment
	csvReader.LazyQuotes = p.config.LazyQuotes
	csvReader.TrimLeadingSpace = p.config.TrimLeadingSpace
	
	rowNum := 0
	
	// Read header if configured
	if p.config.HasHeader {
		header, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("CSV file is empty")
			}
			return fmt.Errorf("failed to read header: %w", err)
		}
		
		// Trim header values
		for i := range header {
			header[i] = strings.TrimSpace(header[i])
		}
		
		p.header = header
		rowNum++
	}
	
	// Read rows
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read row %d: %w", rowNum+1, err)
		}
		
		rowNum++
		
		// Skip empty rows if configured
		if p.config.SkipEmptyRows {
			isEmpty := true
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					isEmpty = false
					break
				}
			}
			if isEmpty {
				continue
			}
		}
		
		// Trim row values
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		
		// Validate row
		for _, validator := range p.config.Validators {
			if err := validator(row, rowNum); err != nil {
				return fmt.Errorf("validation failed for row %d: %w", rowNum, err)
			}
		}
		
		// Call handler
		if err := handler(rowNum, p.header, row); err != nil {
			return fmt.Errorf("handler error for row %d: %w", rowNum, err)
		}
	}
	
	return nil
}

// ParseToSlice parses CSV data and returns all rows as a slice.
func (p *Parser) ParseToSlice(reader io.Reader) ([][]string, error) {
	var rows [][]string
	
	err := p.Parse(reader, func(rowNum int, headers []string, row []string) error {
		rows = append(rows, row)
		return nil
	})
	
	return rows, err
}

// Common validators

// RequiredColumnsValidator validates that required columns are present in the header.
func RequiredColumnsValidator(requiredColumns []string) RowValidator {
	return func(row []string, rowNum int) error {
		// This validator only makes sense for header validation
		// For row validation, you'd need to know the header
		return nil
	}
}

// NonEmptyRowValidator validates that a row is not empty.
func NonEmptyRowValidator() RowValidator {
	return func(row []string, rowNum int) error {
		for i, cell := range row {
			if strings.TrimSpace(cell) == "" {
				return fmt.Errorf("empty cell at column %d", i+1)
			}
		}
		return nil
	}
}

// MinColumnsValidator validates that a row has at least N columns.
func MinColumnsValidator(minCols int) RowValidator {
	return func(row []string, rowNum int) error {
		if len(row) < minCols {
			return fmt.Errorf("row has %d columns, expected at least %d", len(row), minCols)
		}
		return nil
	}
}

// ExactColumnsValidator validates that a row has exactly N columns.
func ExactColumnsValidator(exactCols int) RowValidator {
	return func(row []string, rowNum int) error {
		if len(row) != exactCols {
			return fmt.Errorf("row has %d columns, expected exactly %d", len(row), exactCols)
		}
		return nil
	}
}

