package avatar_test

import (
	"strings"
	"testing"

	"github.com/philiplambok/tudu/pkg/avatar"
)

func TestGravatar_GetAvatarURL(t *testing.T) {
	p := avatar.NewGravatar()
	url := p.GetAvatarURL("test@example.com")
	if !strings.HasPrefix(url, "https://www.gravatar.com/avatar/") {
		t.Errorf("expected Gravatar URL, got %s", url)
	}
}

func TestGravatar_CaseInsensitive(t *testing.T) {
	p := avatar.NewGravatar()
	lower := p.GetAvatarURL("test@example.com")
	upper := p.GetAvatarURL("TEST@EXAMPLE.COM")
	if lower != upper {
		t.Errorf("expected same URL for same email regardless of case: %s != %s", lower, upper)
	}
}

func TestMock_GetAvatarURL(t *testing.T) {
	p := avatar.NewMock()
	url := p.GetAvatarURL("any@email.com")
	if url == "" {
		t.Error("expected non-empty URL from mock provider")
	}
	url2 := p.GetAvatarURL("other@email.com")
	if url != url2 {
		t.Errorf("mock should return same URL for all emails: %s != %s", url, url2)
	}
}
