package catalog

import "strings"

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}
