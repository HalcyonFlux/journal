package server

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// boxStr puts a string in a box
func boxStr(value string) string {
	length := utf8.RuneCountInString(value)
	lines := fmt.Sprintf("+%s+", strings.Repeat("-", length+2))
	content := fmt.Sprintf("| %s |", value)

	return fmt.Sprintf("%s\n%s\n%s", lines, content, lines)
}

// tableStr turns a slice of string slices into a table
func tableStr(table [][]string) string {

	if len(table) == 0 || len(table[0]) == 0 {
		return "No rows found"
	}

	rows := len(table)
	columns := len(table[0])

	// Max widths
	widths := make([]int, columns)
	for _, row := range table {
		for j, value := range row {
			if width := utf8.RuneCountInString(value); width > widths[j] {
				widths[j] = width
			}
		}
	}

	// Row border
	headrowBorder := "+"
	rowBorder := "+"
	for _, width := range widths {
		headrowBorder += strings.Repeat("=", width+2) + "+"
		rowBorder += strings.Repeat("-", width+2) + "+"
	}

	// Table contents
	prettyTable := []string{}
	for i := 0; i < rows; i++ {
		row := table[i]

		rowText := "|"
		for j := 0; j < columns; j++ {
      vallen :=  utf8.RuneCountInString(row[j])
			reps := int((widths[j] + 2 - vallen) / 2)
			space1 := strings.Repeat(" ", reps)
      space2 := strings.Repeat(" ", widths[j] + 2 - vallen - reps)
			rowText += fmt.Sprintf("%s%s%s|", space1, row[j], space2)
		}
		if i == 0 {
			prettyTable = append(prettyTable, headrowBorder, rowText, headrowBorder)
		} else {
			prettyTable = append(prettyTable, rowBorder, rowText, rowBorder)
		}
	}

	return strings.Join(prettyTable, "\n")

}
