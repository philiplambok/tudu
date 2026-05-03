package avatar

type mock struct{}

func NewMock() Provider {
	return &mock{}
}

func (m *mock) GetAvatarURL(_ string) string {
	return "https://api.dicebear.com/7.x/identicon/svg?seed=tudu"
}
