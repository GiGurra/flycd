package globals

import (
	"sync/atomic"
)

// TODO: This entire file is a hack. Remove it and pass the access token around in the context instead

type AtomicString struct {
	value atomic.Value
}

func (s *AtomicString) Load() string {
	v := s.value.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

func (s *AtomicString) Store(v string) {
	s.value.Store(v)
}

var AccessToken AtomicString

func SetAccessToken(token string) {
	AccessToken.Store(token)
}

func GetAccessToken() string {
	return AccessToken.Load()
}
