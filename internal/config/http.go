package config

import "time"

type HTTP struct {
	BaseURL   string        `env:"BASE_URL,expand" envDefault:"/"`
	Address   string        `env:"ADDRESS,expand" envDefault:":3002"`
	Authn     Authn         `envPrefix:"AUTHN_"`
	Session   Session       `envPrefix:"SESSION_"`
	RateLimit HTTPRateLimit `envPrefix:"RATE_LIMIT_"`
}
type Authn struct {
	Providers       AuthProviders `envPrefix:"PROVIDERS_"`
	DefaultAdmins   []string      `env:"DEFAULT_ADMINS" envSeparator:","`
	ActiveByDefault bool          `env:"ACTIVE_BY_DEFAULT" envDefault:"false"`
}

type Session struct {
	Keys   []string `env:"KEYS" envSeparator:","`
	Cookie Cookie   `envPrefix:"COOKIE_"`
}

type Cookie struct {
	Path     string        `env:"PATH" envDefault:"/"`
	HTTPOnly bool          `env:"HTTP_ONLY" envDefault:"true"`
	Secure   bool          `env:"SECURE" envDefault:"false"`
	MaxAge   time.Duration `env:"MAX_AGE" enDefault:"24h"`
}

type AuthProviders struct {
	Google OAuth2Provider `envPrefix:"GOOGLE_"`
	Github OAuth2Provider `envPrefix:"GITHUB_"`
	Gitea  GiteaProvider  `envPrefix:"GITEA_"`
	OIDC   OIDCProvider   `envPrefix:"OIDC_"`
}

type OAuth2Provider struct {
	Key    string   `env:"KEY"`
	Secret string   `env:"SECRET"`
	Scopes []string `env:"SCOPES" envSeparator:"," envDefault:"profile,openid,email"`
}

type OIDCProvider struct {
	OAuth2Provider
	DiscoveryURL string `env:"DISCOVERY_URL"`
	Icon         string `env:"ICON"`
	Label        string `env:"LABEL"`
}

type GiteaProvider struct {
	OAuth2Provider
	TokenURL   string `env:"TOKEN_URL"`
	AuthURL    string `env:"AUTH_URL"`
	ProfileURL string `env:"PROFILE_URL"`
	Label      string `env:"LABEL"`
}

type HTTPRateLimit struct {
	TrustHeaders    bool          `env:"TRUST_HEADERS" envDefault:"true"`
	RequestInterval time.Duration `env:"REQUEST_INTERVAL,expand" envDefault:"1s"`
	RequestMaxBurst int           `env:"REQUEST_MAX_BURST,expand" envDefault:"10"`
	CacheSize       int           `env:"CACHE_SIZE,expand" envDefault:"50"`
	CacheTTL        time.Duration `env:"CACHE_TTL,expand" envDefault:"1h"`
}
