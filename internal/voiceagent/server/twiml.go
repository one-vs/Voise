package server

import (
	"fmt"
	"strings"
)

// TwiMLBuilder helps in building TwiML responses.
type TwiMLBuilder struct {
	sb strings.Builder
}

// NewTwiMLBuilder creates a new TwiMLBuilder.
func NewTwiMLBuilder() *TwiMLBuilder {
	b := &TwiMLBuilder{}
	b.sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Response>\n")
	return b
}

// ConnectStream adds a <Connect><Stream> element.
func (b *TwiMLBuilder) ConnectStream(url string, params map[string]string) *TwiMLBuilder {
	b.sb.WriteString(fmt.Sprintf("  <Connect>\n    <Stream url=\"%s\">\n", url))
	for k, v := range params {
		b.sb.WriteString(fmt.Sprintf("      <Parameter name=\"%s\" value=\"%s\"/>\n", k, v))
	}
	b.sb.WriteString("    </Stream>\n  </Connect>\n")
	return b
}

// Say adds a <Say> element.
func (b *TwiMLBuilder) Say(text string) *TwiMLBuilder {
	b.sb.WriteString(fmt.Sprintf("  <Say>%s</Say>\n", text))
	return b
}

// Build returns the TwiML string.
func (b *TwiMLBuilder) Build() string {
	b.sb.WriteString("</Response>")
	return b.sb.String()
}
