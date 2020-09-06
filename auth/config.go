package auth

import (
	"github.com/urfave/cli/v2"
)

// DefaultCLIFlagsOptions ...
type DefaultCLIFlagsOptions struct {
	Generate  bool
	ExpireSec int
	Issuer    string
	Audience  string
}

// DefaultCLIFlags ...
func DefaultCLIFlags(options *DefaultCLIFlagsOptions) []cli.Flag {

	// Sensible defaults
	if options.ExpireSec == 0 {
		options.ExpireSec = 1 * 24 * 60 * 60 // 1 day
	}

	return []cli.Flag{
		// ... from environment variables
		&cli.StringFlag{
			Name:    "key",
			Aliases: []string{"public-key", "signing-key"},
			EnvVars: []string{"PRIVATE_KEY", "KEY", "SIGNING_KEY"},
			Usage:   "private key to sign the tokens with",
		},
		&cli.StringFlag{
			Name:    "jwks",
			Aliases: []string{"jwks-json", "jwk-set"},
			EnvVars: []string{"JWKS", "JWK_SET", "JWKS_JSON"},
			Usage:   "json encoded jwk set containing the public keys",
		},
		// ... from files
		&cli.PathFlag{
			Name:    "key-file",
			Aliases: []string{"public-key-file", "signing-key-file"},
			EnvVars: []string{"PRIVATE_KEY_FILE", "KEY_FILE", "SIGNING_KEY_FILE"},
			Usage:   "file with private key to sign the tokens with",
		},
		&cli.PathFlag{
			Name:    "jwks-file",
			Aliases: []string{"jwks-json-file", "jwk-set-file"},
			EnvVars: []string{"JWKS_FILE", "JWK_SET_FILE", "JWKS_JSON_FILE"},
			Usage:   "json file with the jwk set containing the public keys",
		},
		&cli.BoolFlag{
			Name:    "generate",
			Value:   options.Generate,
			Aliases: []string{"gen", "create"},
			EnvVars: []string{"GENERATE", "CREATE", "GEN"},
			Usage:   "generate new keys if none were supplied",
		},
		&cli.IntFlag{
			Name:    "expire-sec",
			Value:   options.ExpireSec,
			Aliases: []string{"expire"},
			EnvVars: []string{"EXPIRATION_SEC"},
			Usage:   "number of seconds until a user token expires",
		},
		&cli.StringFlag{
			Name:    "issuer",
			Value:   options.Issuer,
			Aliases: []string{"jwt-issuer"},
			EnvVars: []string{"ISSUER"},
			Usage:   "jwt token issuer",
		},
		&cli.StringFlag{
			Name:    "audience",
			Value:   options.Audience,
			Aliases: []string{"jwt-audience"},
			EnvVars: []string{"AUDIENCE"},
			Usage:   "jwt token audience",
		},
	}
}

// AuthenticatorKeyConfig ...
type AuthenticatorKeyConfig struct {
	Jwks     string
	JwksFile string
	Key      string
	KeyFile  string
	Generate bool
}

// Parse ...
func (c AuthenticatorKeyConfig) Parse(ctx *cli.Context) *AuthenticatorKeyConfig {
	return &AuthenticatorKeyConfig{
		Jwks:     ctx.String("jwks"),
		JwksFile: ctx.String("jwks-file"),
		Key:      ctx.String("key"),
		KeyFile:  ctx.String("key-file"),
		Generate: ctx.Bool("generate"),
	}
}
