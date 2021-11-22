package server

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

var RedisPrefix = "rpc-endpoint:"
var RedisPrefixTxSentToRelay = RedisPrefix + "tx-sent-to-relay:"
var RedisExpiryTxSentToRelay = time.Duration(24 * time.Hour) // 1 day

var RedisPrefixTxHashForSenderAndNonce = RedisPrefix + "txsender-and-nonce-to-txhash:"
var RedisExpiryTxHashForSenderAndNonce = time.Duration(24 * time.Hour) // 1 day

var RedisPrefixNonceFixForAccount = RedisPrefix + "txsender-with-nonce-fix:"
var RedisExpiryNonceFixForAccount = time.Duration(2 * time.Hour)

func RedisKeyTxSentToRelay(txHash string) string {
	return RedisPrefixTxSentToRelay + strings.ToLower(txHash)
}

func RedisKeyTxHashForSenderAndNonce(txFrom string, nonce uint64) string {
	return fmt.Sprintf("%s%s_%d", RedisPrefixTxHashForSenderAndNonce, strings.ToLower(txFrom), nonce)
}

func RedisKeyNonceFixForAccount(txFrom string) string {
	return RedisPrefixNonceFixForAccount + strings.ToLower(txFrom)
}

type RedisState struct {
	RedisClient *redis.Client
}

func NewRedisState(redisUrl string) (*RedisState, error) {
	// Setup redis client and check connection
	redisClient := redis.NewClient(&redis.Options{Addr: redisUrl})

	// Try to get a key to see if there's an error with the connection
	if err := redisClient.Get(context.Background(), "somekey").Err(); err != nil && err != redis.Nil {
		return nil, errors.Wrap(err, "redis init error")
	}

	// Create and return the RedisState
	return &RedisState{
		RedisClient: redisClient,
	}, nil
}

func (s *RedisState) SetTxSentToRelay(txHash string) error {
	key := RedisKeyTxSentToRelay(txHash)
	err := s.RedisClient.Set(context.Background(), key, time.Now().UTC().Unix(), RedisExpiryTxSentToRelay).Err()
	return err
}

func (s *RedisState) GetTxSentToRelay(txHash string) (timeSent time.Time, found bool, err error) {
	key := RedisKeyTxSentToRelay(txHash)
	val, err := s.RedisClient.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return time.Time{}, false, nil // just not found
	} else if err != nil {
		return time.Time{}, false, err // found but error
	}

	timestampInt, err := strconv.Atoi(val)
	if err != nil {
		return time.Time{}, true, err // found but error
	}

	t := time.Unix(int64(timestampInt), 0)
	return t, true, nil
}

func (s *RedisState) SetTxHashForSenderAndNonce(txFrom string, nonce uint64, txHash string) error {
	key := RedisKeyTxHashForSenderAndNonce(txFrom, nonce)
	err := s.RedisClient.Set(context.Background(), key, strings.ToLower(txHash), RedisExpiryTxHashForSenderAndNonce).Err()
	return err
}

func (s *RedisState) GetTxHashForSenderAndNonce(txFrom string, nonce uint64) (val string, found bool, err error) {
	key := RedisKeyTxHashForSenderAndNonce(txFrom, nonce)
	val, err = s.RedisClient.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return "", false, nil // not found
	} else if err != nil {
		return "", false, err
	}

	return val, true, nil
}

func (s *RedisState) SetNonceFixForAccount(txFrom string, numTimesSent uint64) error {
	key := RedisKeyNonceFixForAccount(txFrom)
	err := s.RedisClient.Set(context.Background(), key, numTimesSent, RedisExpiryNonceFixForAccount).Err()
	return err
}

func (s *RedisState) DelNonceFixForAccount(txFrom string) error {
	key := RedisKeyNonceFixForAccount(txFrom)
	err := s.RedisClient.Del(context.Background(), key).Err()
	return err
}

func (s *RedisState) GetNonceFixForAccount(txFrom string) (numTimesSent uint64, found bool, err error) {
	key := RedisKeyNonceFixForAccount(txFrom)
	val, err := s.RedisClient.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return 0, false, nil // not found
	} else if err != nil {
		return 0, false, err
	}

	numTimesSent, err = strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, true, err
	}
	return numTimesSent, true, nil
}
