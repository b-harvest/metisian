package util

import (
	"fmt"
	"strings"
)

func FormatSliceToNLStr(slice []string) string {
	var formatted []string
	for _, str := range slice {
		formatted = append(formatted, fmt.Sprintf("- \"%s\"", str))
	}
	return strings.Join(formatted, "\n")
}
