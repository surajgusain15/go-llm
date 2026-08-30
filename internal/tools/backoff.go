package tools

import "time"

type BackoffPolicy func(attempt int) time.Duration

func ExponentialBackoff(
	base time.Duration,
	max time.Duration,
) BackoffPolicy {

	return func(attempt int) time.Duration {
		if attempt <= 0 || base <= 0 {
			return 0
		}

		delay := base

		for i := 1; i < attempt; i++ {
			if max > 0 && delay >= max {
				return max
			}

			if max > 0 && delay > max/2 {
				return max
			}

			delay *= 2
		}

		if max > 0 && delay > max {
			return max
		}

		return delay
	}
}

func ExponentialBackoffWithJitter(
	base time.Duration,
	max time.Duration,
	jitter time.Duration,
) BackoffPolicy {

	exponential := ExponentialBackoff(
		base,
		max,
	)

	return func(attempt int) time.Duration {
		delay := exponential(attempt)

		if delay <= 0 || jitter <= 0 {
			return delay
		}

		offset := time.Duration(
			time.Now().UnixNano()%
				int64(jitter*2+1),
		) - jitter

		delay += offset

		if delay < 0 {
			return 0
		}

		if max > 0 && delay > max {
			return max
		}

		return delay
	}
}
