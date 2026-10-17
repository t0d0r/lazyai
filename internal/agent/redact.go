package agent

import (
	"regexp"
	"strings"
)

var secretName = regexp.MustCompile(`(?i)(api[-_]?key|token|secret|passw(or)?d|credential|auth)`)

const redacted = "[redacted]"

// RedactArgs hides values that look like credentials so a command line can
// be shown on screen or written as JSON. It catches "--api-key=X",
// "--token X" and "SECRET_TOKEN=X".
func RedactArgs(args string) string {
	tokens := strings.Fields(args)
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		name, _, hasValue := strings.Cut(t, "=")
		switch {
		case hasValue && secretName.MatchString(name):
			tokens[i] = name + "=" + redacted
		case strings.HasPrefix(t, "-") && secretName.MatchString(t) &&
			i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-"):
			tokens[i+1] = redacted
			i++
		}
	}
	return strings.Join(tokens, " ")
}
