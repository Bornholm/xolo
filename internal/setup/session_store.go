package setup

import (
	"context"
	"net/http"

	"github.com/bornholm/xolo/internal/config"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

var getSessionStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (sessions.Store, error) {
	keyPairs := make([][]byte, 0)
	if len(conf.HTTP.Session.Keys) == 0 {
		key, err := getRandomBytes(32)
		if err != nil {
			return nil, errors.Wrap(err, "could not generate cookie signing key")
		}

		keyPairs = append(keyPairs, key)
	} else {
		for _, k := range conf.HTTP.Session.Keys {
			keyPairs = append(keyPairs, []byte(k))
		}
	}

	sessionStore := sessions.NewCookieStore(keyPairs...)

	sessionStore.MaxAge(int(conf.HTTP.Session.Cookie.MaxAge.Seconds()))
	sessionStore.Options.Path = string(conf.HTTP.Session.Cookie.Path)
	sessionStore.Options.HttpOnly = bool(conf.HTTP.Session.Cookie.HTTPOnly)
	sessionStore.Options.Secure = conf.HTTP.Session.Cookie.Secure
	sessionStore.Options.SameSite = http.SameSiteLaxMode

	return sessionStore, nil
})
