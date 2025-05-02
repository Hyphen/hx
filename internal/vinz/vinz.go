package vinz

type Key struct {
	SecretKeyId int64  `json:"secret_key_id"`
	SecretKey   string `json:"secret_key"`
}

type KeyResponse struct {
	Key Key `json:"key"`
}
