package avatar

import (
	"crypto/md5"
	"fmt"
	"strings"
)

type gravatar struct{}

func NewGravatar() Provider {
	return &gravatar{}
}

func (g *gravatar) GetAvatarURL(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=identicon", hash)
}
