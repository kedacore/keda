package kql

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// RequiresQuoting checks whether a given string is an identifier
func RequiresQuoting(value string) bool {
	if value == "" {
		return false
	}

	for _, c := range value {
		if c == '_' {
			continue
		}

		if c > unicode.MaxLatin1 || !(unicode.IsLetter(c) || unicode.IsDigit(c)) {
			return true
		}
	}
	return false
}

func QuoteString(value string, hidden bool) string {
	if value == "" {
		return value
	}

	var literal strings.Builder

	if hidden {
		literal.WriteString("h")
	}
	literal.WriteString("\"")
	for _, c := range value {
		switch c {
		case '\'':
			literal.WriteString("\\'")

		case '"':
			literal.WriteString("\\\"")

		case '\\':
			literal.WriteString("\\\\")

		case '\x00':
			literal.WriteString("\\0")

		case '\a':
			literal.WriteString("\\a")

		case '\b':
			literal.WriteString("\\b")

		case '\f':
			literal.WriteString("\\f")

		case '\n':
			literal.WriteString("\\n")

		case '\r':
			literal.WriteString("\\r")

		case '\t':
			literal.WriteString("\\t")

		case '\v':
			literal.WriteString("\\v")

		default:
			if !ShouldBeEscaped(c) {
				literal.WriteString(string(c))
			} else {
				literal.WriteString(fmt.Sprintf("\\u%04x", c))
			}

		}
	}
	literal.WriteString("\"")

	return literal.String()
}

// ShouldBeEscaped Checks whether a rune should be escaped or not based on it's type.
func ShouldBeEscaped(c int32) bool {
	if c <= unicode.MaxLatin1 {
		return unicode.IsControl(c)
	}
	return true
}

func FormatTimespan(duration time.Duration) string {
	// Calculate the number of days in the duration
	days := duration / (24 * time.Hour)

	// Calculate the remaining time after subtracting the days
	remaining := duration - days*24*time.Hour

	daysStr := ""
	if days > 0 {
		daysStr = fmt.Sprintf("%d.", days)
	}

	// Use the `fmt.Sprintf()` function to format the duration as a string
	return fmt.Sprintf("%s%02d:%02d:%02d.%07d",
		daysStr,
		int(remaining.Hours()),
		int(remaining.Minutes())%60,
		int(remaining.Seconds())%60,
		int((remaining.Nanoseconds())%1000000000/100))
}

func FormatDatetime(datetime time.Time) string {
	return datetime.Format("2006-01-02T15:04:05.9999999Z07:00")
}
