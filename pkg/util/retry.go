package util

import (
	"context"
	"github.com/avast/retry-go/v4"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"time"
)

func RetryWithBackOff(ctx context.Context,
	duration time.Duration, factor, jitter float64, steps int, cap time.Duration,
	fn func() error, shouldRetry func(error) bool, onRetry func(n uint, err error)) error {
	var opts []retry.Option
	opts = append(opts, retry.LastErrorOnly(true))

	opts = append(opts, retry.Context(ctx))
	if shouldRetry != nil {
		opts = append(opts, retry.RetryIf(shouldRetry))
	}

	if onRetry != nil {
		opts = append(opts, retry.OnRetry(onRetry))
	}

	b := wait.Backoff{
		Duration: duration,
		Factor:   factor,
		Jitter:   jitter,
		Steps:    steps,
		Cap:      cap,
	}

	opts = append(opts, retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
		next := b.Step()
		return next
	}))

	err := retry.Do(func() error {
		return fn()
	}, opts...)
	if err != nil {
		klog.Errorf("retry do err: %v", err)
		return err
	}

	return nil
}
