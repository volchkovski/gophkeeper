// Package models provides common data types shared between client and server.
package models

// LoginPassword represents credentials for a website or service.
type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextData represents arbitrary text data.
type TextData struct {
	Content string `json:"content"`
}

// BinaryData represents binary file data.
type BinaryData struct {
	Data     []byte `json:"data"`
	Filename string `json:"filename"`
}

// CardData represents credit/debit card information.
type CardData struct {
	Number     string `json:"number"`
	Holder     string `json:"holder"`
	ExpiryDate string `json:"expiry_date"`
	CVV        string `json:"cvv"`
}

// Metadata represents additional information about a secret.
type Metadata struct {
	Website      string            `json:"website,omitempty"`
	Notes        string            `json:"notes,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

