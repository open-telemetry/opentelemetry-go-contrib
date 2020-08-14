package gomemcache

import (
	"github.com/bradfitz/gomemcache/memcache"
	"google.golang.org/grpc/codes"
)

// maps memcache error to appropriate error code; otherwise returns status OK
func memcacheErrToStatusCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	switch err {
	case memcache.ErrCacheMiss, memcache.ErrNotStored, memcache.ErrNoStats:
		return codes.NotFound
	case memcache.ErrCASConflict:
		return codes.AlreadyExists
	case memcache.ErrServerError:
		return codes.Internal
	case memcache.ErrMalformedKey:
		return codes.InvalidArgument
	default:
		return codes.Unknown
	}
}
