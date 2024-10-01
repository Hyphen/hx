package zelda

import "time"

// Code

type Code struct {
	ID             *string  `json:"id,omitempty"`
	LongURL        string   `json:"long_url"`
	Domain         string   `json:"domain"`
	Code           *string  `json:"code,omitempty"`
	Title          *string  `json:"title,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	OrganizationID string   `json:"organizationId,omitempty"` // DEVNOTE: The API is inconsistent and expecting organizationId, not organizaiton_id (unfortunately)
}

//Domain

type DomainInfo struct {
	ID        string     `json:"id"`
	Domain    string     `json:"domain"`
	Status    string     `json:"status"`
	DNSStatus string     `json:"dnsStatus"`
	SSLStatus string     `json:"sslStatus"`
	CreatedAt *time.Time `json:"createdAt"`
}

//QR

type QR struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	QRCode string `json:"qrCode"`
	QRLink string `json:"qrLink"`
}
