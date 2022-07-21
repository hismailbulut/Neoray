package logger

import (
	"strings"

	"github.com/hismailbulut/Neoray/pkg/common"
)

type Table struct {
	headers []string
	rows    [][]string
}

func NewTable(headers []string) *Table {
	return &Table{
		headers: headers,
	}
}

func (t *Table) AddRow(row []string) {
	if len(t.headers) != len(row) {
		panic("row length does not match headers length")
	}
	t.rows = append(t.rows, row)
}

func (t *Table) Render(s *strings.Builder) {
	// Calculate cell lengths
	maxCellLengths := make([]int, len(t.headers))
	for _, row := range t.rows {
		for i, cell := range row {
			if len(cell) > maxCellLengths[i] {
				maxCellLengths[i] = len(cell)
			}
		}
	}
	for i := range t.headers {
		maxCellLengths[i] = common.Max(len(t.headers[i]), maxCellLengths[i])
	}
	// Write separator
	for i := range maxCellLengths {
		if i == 0 {
			s.WriteRune('+')
		}
		s.WriteString(strings.Repeat("-", maxCellLengths[i]+2))
		s.WriteRune('+')
	}
	s.WriteRune('\n')
	// Write headers
	for i := range t.headers {
		if i == 0 {
			s.WriteString("| ")
		}
		// Center header
		spaceCount := maxCellLengths[i] - len(t.headers[i])
		halfSpaceCount := spaceCount / 2
		additionalSpaceCount := spaceCount % 2
		s.WriteString(strings.Repeat(" ", halfSpaceCount))
		s.WriteString(t.headers[i])
		s.WriteString(strings.Repeat(" ", halfSpaceCount+additionalSpaceCount))
		if i != len(t.headers)-1 {
			s.WriteString(" | ")
		} else {
			s.WriteString(" |\n")
		}
	}
	// Write separator
	for i := range maxCellLengths {
		if i == 0 {
			s.WriteRune('+')
		}
		s.WriteString(strings.Repeat("-", maxCellLengths[i]+2))
		s.WriteRune('+')
	}
	s.WriteRune('\n')
	// Write cells
	for _, row := range t.rows {
		for i, cell := range row {
			if i == 0 {
				s.WriteString("| ")
			}
			s.WriteString(cell)
			s.WriteString(strings.Repeat(" ", maxCellLengths[i]-len(cell)))
			if i != len(row)-1 {
				s.WriteString(" | ")
			} else {
				s.WriteString(" |\n")
			}
		}
	}
	// Write separator
	for i := range maxCellLengths {
		if i == 0 {
			s.WriteRune('+')
		}
		s.WriteString(strings.Repeat("-", maxCellLengths[i]+2))
		s.WriteRune('+')
	}
	s.WriteRune('\n')
}
