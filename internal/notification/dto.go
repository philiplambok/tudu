package notification

import "time"

type tokenRequestDTO struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

type tokenResponseDTO struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type SendRequestDTO struct {
	RecipientID string            `json:"recipient_id"`
	Channel     string            `json:"channel"`
	TemplateID  string            `json:"template_id"`
	Payload     map[string]string `json:"payload"`
}

type SendResponseDTO struct {
	NotificationID string    `json:"notification_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type errorResponseDTO struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
