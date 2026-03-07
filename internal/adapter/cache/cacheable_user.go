package cache

import (
	"fmt"
	"strings"

	"github.com/bornholm/xolo/internal/core/model"
)

type CacheableUser struct {
	model.User
}

// CacheKeys implements [Cacheable].
func (u *CacheableUser) CacheKeys() []string {
	return []string{
		getUserProviderSubjectCacheKey(u.Provider(), u.Subject()),
		string(u.ID()),
	}
}

func NewCacheableUser(user model.User) *CacheableUser {
	return &CacheableUser{user}
}

var (
	_ model.User = &CacheableUser{}
	_ Cacheable  = &CacheableUser{}
)

func getUserProviderSubjectCacheKey(provider string, subject string) string {
	return getCompositeCacheKey(provider, subject)
}

func getCompositeCacheKey(parts ...any) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString("|")
		}
		sb.WriteString(fmt.Sprintf("%s", p))
	}
	return sb.String()
}
