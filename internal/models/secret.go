package models

type Secret struct {
	SecretKeyId     int64  `json:"secret_key_id"`
	Base64SecretKey string `json:"secret_key"`
}
