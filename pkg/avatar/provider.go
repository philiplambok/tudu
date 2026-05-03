package avatar

// Provider generates an avatar URL for a given email address.
// Selected at startup: NewGravatar for production, NewMock for local/sandbox.
type Provider interface {
	GetAvatarURL(email string) string
}
