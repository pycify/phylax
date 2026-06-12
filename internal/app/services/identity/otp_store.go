package identity

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrOTPNotFound = errors.New("otp not found or expired")

type OTPStore interface {
	Set(ctx context.Context, namespace string, key string, code string, ttl time.Duration) error
	Verify(ctx context.Context, namespace string, key string, code string) error
}

type MemoryOTPStore struct {
	mu     sync.Mutex
	values map[string]memoryOTP
}

type memoryOTP struct {
	code      string
	expiresAt time.Time
}

func NewMemoryOTPStore() *MemoryOTPStore {
	return &MemoryOTPStore{values: map[string]memoryOTP{}}
}

func (s *MemoryOTPStore) Set(_ context.Context, namespace string, key string, code string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[otpKey(namespace, key)] = memoryOTP{code: code, expiresAt: time.Now().UTC().Add(ttl)}
	return nil
}

func (s *MemoryOTPStore) Verify(_ context.Context, namespace string, key string, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[otpKey(namespace, key)]
	if !ok || time.Now().UTC().After(value.expiresAt) || value.code != code {
		return ErrOTPNotFound
	}
	delete(s.values, otpKey(namespace, key))
	return nil
}

type RedisOTPStore struct {
	client *redis.Client
}

func NewRedisOTPStore(redisURL string) (*RedisOTPStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &RedisOTPStore{client: redis.NewClient(options)}, nil
}

func (s *RedisOTPStore) Set(ctx context.Context, namespace string, key string, code string, ttl time.Duration) error {
	return s.client.Set(ctx, otpKey(namespace, key), code, ttl).Err()
}

func (s *RedisOTPStore) Verify(ctx context.Context, namespace string, key string, code string) error {
	stored, err := s.client.Get(ctx, otpKey(namespace, key)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrOTPNotFound
	}
	if err != nil {
		return err
	}
	if stored != code {
		return ErrOTPNotFound
	}
	return s.client.Del(ctx, otpKey(namespace, key)).Err()
}

func otpKey(namespace string, key string) string {
	return "phylax:otp:" + namespace + ":" + key
}
