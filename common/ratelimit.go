package common

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ipLimiter holds per-IP rate limiters for both unary and streaming RPCs.
type ipLimiter struct {
	unary    *rate.Limiter
	stream   *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting for gRPC calls.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter

	unaryRate  rate.Limit
	unaryBurst int

	streamRate  rate.Limit
	streamBurst int
}

// NewRateLimiter creates a new rate limiter and starts background cleanup.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters:    make(map[string]*ipLimiter),
		unaryRate:   50, // 50 requests/second per IP
		unaryBurst:  100,
		streamRate:  10, // 10 new streams/second per IP
		streamBurst: 20,
	}
	go rl.cleanup()
	return rl
}

// cleanup removes stale entries every 5 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, l := range rl.limiters {
			if time.Since(l.lastSeen) > 10*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getLimiter returns the per-IP limiter, creating one if needed.
func (rl *RateLimiter) getLimiter(addr string) *ipLimiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.limiters[addr]
	if !ok {
		l = &ipLimiter{
			unary:  rate.NewLimiter(rl.unaryRate, rl.unaryBurst),
			stream: rate.NewLimiter(rl.streamRate, rl.streamBurst),
		}
		rl.limiters[addr] = l
	}
	l.lastSeen = time.Now()
	return l
}

func peerAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}

// UnaryInterceptor returns a gRPC unary server interceptor that rate limits per IP.
func (rl *RateLimiter) UnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	addr := peerAddr(ctx)
	l := rl.getLimiter(addr)
	if !l.unary.Allow() {
		return nil, status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded for %s", addr)
	}
	return handler(ctx, req)
}

// StreamInterceptor returns a gRPC stream server interceptor that rate limits per IP.
func (rl *RateLimiter) StreamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	addr := peerAddr(ss.Context())
	l := rl.getLimiter(addr)
	if !l.stream.Allow() {
		return status.Errorf(codes.ResourceExhausted,
			"stream rate limit exceeded for %s", addr)
	}
	return handler(srv, ss)
}
