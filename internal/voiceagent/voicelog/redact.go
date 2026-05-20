package voicelog

import (
	"regexp"
)

var (
	creditCardRegex = regexp.MustCompile(`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`)
	ssnRegex        = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
)

// RedactPII masks sensitive information in a string.
func RedactPII(text string) string {
	text = creditCardRegex.ReplaceAllString(text, "**** **** **** ****")
	text = ssnRegex.ReplaceAllString(text, "***-**-****")
	return text
}
