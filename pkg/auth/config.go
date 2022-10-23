package auth

// KeyConfig configures the keys that will be used for authentication
type KeyConfig struct {
	Jwks     string
	JwksFile string
	Key      string
	KeyFile  string
	Generate bool
}
