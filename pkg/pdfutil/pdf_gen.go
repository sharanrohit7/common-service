package pdfutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/jung-kurt/gofpdf"
)

// PDFGenerator provides utilities for generating PDF documents.
type PDFGenerator struct {
	pdf *gofpdf.Fpdf
}

// NewPDFGenerator creates a new PDF generator.
func NewPDFGenerator() *PDFGenerator {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	return &PDFGenerator{pdf: pdf}
}

// SetFont sets the font for text rendering.
func (g *PDFGenerator) SetFont(family, style string, size float64) {
	g.pdf.SetFont(family, style, size)
}

// AddText adds text to the PDF at the specified position.
func (g *PDFGenerator) AddText(x, y float64, text string) {
	g.pdf.SetXY(x, y)
	g.pdf.Cell(0, 10, text)
}

// AddLine adds a line to the PDF.
func (g *PDFGenerator) AddLine(x1, y1, x2, y2 float64) {
	g.pdf.Line(x1, y1, x2, y2)
}

// AddTable adds a table to the PDF.
func (g *PDFGenerator) AddTable(headers []string, rows [][]string, x, y, cellWidth, cellHeight float64) {
	currentY := y
	
	// Draw headers
	g.pdf.SetFont("Arial", "B", 12)
	for i, header := range headers {
		g.pdf.SetXY(x+float64(i)*cellWidth, currentY)
		g.pdf.Cell(cellWidth, cellHeight, header)
	}
	currentY += cellHeight
	
	// Draw rows
	g.pdf.SetFont("Arial", "", 10)
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(headers) {
				break
			}
			g.pdf.SetXY(x+float64(i)*cellWidth, currentY)
			g.pdf.Cell(cellWidth, cellHeight, cell)
		}
		currentY += cellHeight
	}
}

// SaveToFile saves the PDF to a file.
func (g *PDFGenerator) SaveToFile(filename string) error {
	return g.pdf.OutputFileAndClose(filename)
}

// WriteToWriter writes the PDF to an io.Writer.
func (g *PDFGenerator) WriteToWriter(w io.Writer) error {
	return g.pdf.Output(w)
}

// GetBytes returns the PDF as a byte slice.
func (g *PDFGenerator) GetBytes() ([]byte, error) {
	var buf bytes.Buffer
	err := g.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GeneratePayslip generates a sample payslip PDF.
func GeneratePayslip(employeeName, employeeID, period string, grossPay, deductions, netPay float64) ([]byte, error) {
	pdf := NewPDFGenerator()
	pdf.SetFont("Arial", "B", 16)
	pdf.AddText(10, 20, "PAYSLIP")
	
	pdf.SetFont("Arial", "", 12)
	pdf.AddText(10, 35, fmt.Sprintf("Employee: %s", employeeName))
	pdf.AddText(10, 42, fmt.Sprintf("Employee ID: %s", employeeID))
	pdf.AddText(10, 49, fmt.Sprintf("Period: %s", period))
	
	pdf.AddLine(10, 60, 200, 60)
	
	pdf.AddText(10, 70, fmt.Sprintf("Gross Pay: $%.2f", grossPay))
	pdf.AddText(10, 77, fmt.Sprintf("Deductions: $%.2f", deductions))
	pdf.AddText(10, 84, fmt.Sprintf("Net Pay: $%.2f", netPay))
	
	return pdf.GetBytes()
}

// GenerateReport generates a sample report PDF with a table.
func GenerateReport(title string, headers []string, rows [][]string) ([]byte, error) {
	pdf := NewPDFGenerator()
	pdf.SetFont("Arial", "B", 16)
	pdf.AddText(10, 20, title)
	
	pdf.AddLine(10, 30, 200, 30)
	
	if len(headers) > 0 && len(rows) > 0 {
		cellWidth := 190.0 / float64(len(headers))
		pdf.AddTable(headers, rows, 10, 40, cellWidth, 8)
	}
	
	return pdf.GetBytes()
}

