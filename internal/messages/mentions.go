package messages

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var mentionRe = regexp.MustCompile(`(?:^|[^\p{L}\p{N}_])@([\p{L}\p{N}_.\-]{3,25})`)

func parseMentions(text string) []string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)

	seen := make(map[string]bool)
	logins := make([]string, 0, len(matches))
	for _, m := range matches {
		login := strings.TrimRight(m[1], ".")
		if utf8.RuneCountInString(login) < 3 || seen[login] {
			continue
		}
		seen[login] = true
		logins = append(logins, login)
	}

	return logins
}
