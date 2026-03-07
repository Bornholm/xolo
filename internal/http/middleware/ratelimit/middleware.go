package ratelimit

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

func Middleware(trustHeaders bool, interval time.Duration, maxBurst int, cacheSize int, ttl time.Duration) func(http.Handler) http.Handler {
	cache := expirable.NewLRU[string, *rate.Limiter](cacheSize, nil, ttl)

	getLimiter := func(remoteAddr string) *rate.Limiter {
		limiter, exists := cache.Get(remoteAddr)
		if !exists {
			limiter = rate.NewLimiter(rate.Every(interval), maxBurst)
			cache.Add(remoteAddr, limiter)
		}

		return limiter
	}

	getRemoteAddr := func(r *http.Request) string {
		if trustHeaders {
			xff := r.Header.Get("X-Forwarded-For")
			if xff != "" {
				ips := strings.Split(xff, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}

			xri := r.Header.Get("X-Real-Ip")
			if xri != "" {
				return xri
			}
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}

		return ip
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteAddr := getRemoteAddr(r)
			limiter := getLimiter(remoteAddr)

			reservation := limiter.Reserve()
			if !reservation.OK() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			if reservation.Delay() > 0 {
				reservation.Cancel()

				w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(reservation.Delay().Seconds()))))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxBurst))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", limiter.Tokens()))
			tokens := limiter.Tokens()
			if tokens < float64(maxBurst) {
				secondsToReset := (float64(maxBurst) - tokens) / float64(maxBurst)
				resetTime := time.Now().Add(time.Duration(secondsToReset) * time.Second)
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			} else {
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Unix(), 10))
			}

			next.ServeHTTP(w, r)
		})
	}
}
