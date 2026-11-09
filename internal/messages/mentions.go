package messages

import (
	"context"
	"fmt"
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

func (s *Service) addMentions(ctx context.Context, m *MessageModel) error {
	logins := parseMentions(m.Message)
	if len(logins) == 0 {
		return nil
	}

	ids, err := s.repository.AddMentions(ctx, m.ID, m.ChatID, m.UserID, logins)
	if err != nil {
		return fmt.Errorf(
			"add mentions: %w",
			err,
		)
	}

	m.Mentions = ids
	return nil
}
