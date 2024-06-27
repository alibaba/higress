package options

import (
	"time"
)

// Cookie contains configuration options relating to Cookie configuration
type Cookie struct {
	Name           string        `mapstructure:"cookie_name"`
	Secret         string        `mapstructure:"cookie_secret"`
	Domains        []string      `mapstructure:"cookie_domains"`
	Path           string        `mapstructure:"cookie_path"`
	Expire         time.Duration `mapstructure:"cookie_expire"`
	Refresh        time.Duration `mapstructure:"cookie_refresh"`
	Secure         bool          `mapstructure:"cookie_secure"`
	HTTPOnly       bool          `mapstructure:"cookie_httponly"`
	SameSite       string        `mapstructure:"cookie_samesite"`
	CSRFPerRequest bool          `mapstructure:"cookie_csrf_per_request"`
	CSRFExpire     time.Duration `mapstructure:"cookie_csrf_expire"`
}

// cookieDefaults creates a Cookie populating each field with its default value
func cookieDefaults() Cookie {
	return Cookie{
		Name:           "_oauth2_proxy",
		Secret:         "",
		Domains:        nil,
		Path:           "/",
		Expire:         time.Duration(168) * time.Hour,
		Refresh:        time.Duration(0),
		Secure:         true,
		HTTPOnly:       true,
		SameSite:       "",
		CSRFPerRequest: false,
		CSRFExpire:     time.Duration(15) * time.Minute,
	}
}
