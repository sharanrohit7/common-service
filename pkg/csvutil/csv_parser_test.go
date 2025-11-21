package csvutil

import (
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	csvData := `name,age,city
John,30,New York
Jane,25,San Francisco
Bob,35,Chicago`

	parser := NewParser(DefaultParserConfig())
	reader := strings.NewReader(csvData)
	
	var rows [][]string
	err := parser.Parse(reader, func(rowNum int, headers []string, row []string) error {
		if rowNum == 1 {
			if len(headers) != 3 {
				t.Errorf("Expected 3 headers, got %d", len(headers))
			}
		}
		rows = append(rows, row)
		return nil
	})
	
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}
	
	if rows[0][0] != "John" {
		t.Errorf("Expected first row first column to be 'John', got '%s'", rows[0][0])
	}
}

func TestParser_ParseToSlice(t *testing.T) {
	csvData := `name,age
John,30
Jane,25`

	parser := NewParser(DefaultParserConfig())
	reader := strings.NewReader(csvData)
	
	rows, err := parser.ParseToSlice(reader)
	if err != nil {
		t.Fatalf("ParseToSlice failed: %v", err)
	}
	
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rows))
	}
}

func TestParser_Validation(t *testing.T) {
	csvData := `name,age
John,30
,25`

	config := DefaultParserConfig()
	config.Validators = []RowValidator{
		NonEmptyRowValidator(),
	}
	
	parser := NewParser(config)
	reader := strings.NewReader(csvData)
	
	err := parser.Parse(reader, func(rowNum int, headers []string, row []string) error {
		return nil
	})
	
	if err == nil {
		t.Error("Expected validation error for empty cell")
	}
}

func TestParser_MinColumns(t *testing.T) {
	csvData := `name,age,city
John,30`

	config := DefaultParserConfig()
	config.Validators = []RowValidator{
		MinColumnsValidator(3),
	}
	
	parser := NewParser(config)
	reader := strings.NewReader(csvData)
	
	err := parser.Parse(reader, func(rowNum int, headers []string, row []string) error {
		return nil
	})
	
	if err == nil {
		t.Error("Expected validation error for insufficient columns")
	}
}

func TestParser_EmptyFile(t *testing.T) {
	csvData := ``
	
	parser := NewParser(DefaultParserConfig())
	reader := strings.NewReader(csvData)
	
	err := parser.Parse(reader, func(rowNum int, headers []string, row []string) error {
		return nil
	})
	
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

