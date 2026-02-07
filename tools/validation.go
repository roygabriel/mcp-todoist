package tools

import (
	"fmt"
	"strings"
)

// ValidateID checks that an ID parameter is safe for use in URL paths.
// It rejects empty values, path traversal sequences, and control characters.
func ValidateID(id, paramName string) error {
	if id == "" {
		return fmt.Errorf("%s is required", paramName)
	}
	if strings.Contains(id, "..") {
		return fmt.Errorf("%s contains invalid characters", paramName)
	}
	if strings.Contains(id, "/") {
		return fmt.Errorf("%s contains invalid characters", paramName)
	}
	for _, c := range id {
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf("%s contains invalid characters", paramName)
		}
	}
	return nil
}
