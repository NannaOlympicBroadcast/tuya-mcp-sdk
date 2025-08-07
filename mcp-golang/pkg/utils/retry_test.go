package utils

import (
	"errors"
	"testing"
	"time"
)

func TestRetryWithBackoff_SuccessOnFirstAttempt(t *testing.T) {
	attempts := 3
	initialDelay := 10 * time.Millisecond
	maxDelay := 100 * time.Millisecond

	called := false
	fn := func() error {
		called = true
		return nil
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err != nil {
		t.Errorf("期望成功，但得到错误: %v", err)
	}

	if !called {
		t.Error("期望函数被调用，但没有被调用")
	}
}

func TestRetryWithBackoff_SuccessOnRetry(t *testing.T) {
	attempts := 3
	initialDelay := 10 * time.Millisecond
	maxDelay := 100 * time.Millisecond

	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("临时错误")
		}
		return nil
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err != nil {
		t.Errorf("期望成功，但得到错误: %v", err)
	}

	if callCount != 2 {
		t.Errorf("期望函数被调用2次，但实际调用了%d次", callCount)
	}
}

func TestRetryWithBackoff_AllAttemptsFail(t *testing.T) {
	attempts := 3
	initialDelay := 10 * time.Millisecond
	maxDelay := 100 * time.Millisecond

	callCount := 0
	expectedError := errors.New("持续错误")
	fn := func() error {
		callCount++
		return expectedError
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err == nil {
		t.Error("期望失败，但没有返回错误")
	}

	if err.Error() != "retry failed" {
		t.Errorf("期望错误消息为 'retry failed'，但得到: %v", err)
	}

	if callCount != attempts {
		t.Errorf("期望函数被调用%d次，但实际调用了%d次", attempts, callCount)
	}
}

func TestRetryWithBackoff_ZeroAttempts(t *testing.T) {
	attempts := 0
	initialDelay := 10 * time.Millisecond
	maxDelay := 100 * time.Millisecond

	called := false
	fn := func() error {
		called = true
		return errors.New("错误")
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err == nil {
		t.Error("期望失败，但没有返回错误")
	}

	if called {
		t.Error("期望函数不被调用，但被调用了")
	}
}

func TestRetryWithBackoff_OneAttempt(t *testing.T) {
	attempts := 1
	initialDelay := 10 * time.Millisecond
	maxDelay := 100 * time.Millisecond

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("错误")
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err == nil {
		t.Error("期望失败，但没有返回错误")
	}

	if callCount != 1 {
		t.Errorf("期望函数被调用1次，但实际调用了%d次", callCount)
	}
}

func TestRetryWithBackoff_DelayProgression(t *testing.T) {
	attempts := 4
	initialDelay := 10 * time.Millisecond
	maxDelay := 50 * time.Millisecond

	callCount := 0
	startTime := time.Now()

	fn := func() error {
		callCount++
		return errors.New("错误")
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err == nil {
		t.Error("期望失败，但没有返回错误")
	}

	elapsed := time.Since(startTime)

	// 验证总耗时应该大于初始延迟的累加
	expectedMinDelay := initialDelay + initialDelay*2 + maxDelay // 第1次重试: initialDelay, 第2次重试: initialDelay*2, 第3次重试: maxDelay
	if elapsed < expectedMinDelay {
		t.Errorf("期望至少延迟 %v，但实际延迟了 %v", expectedMinDelay, elapsed)
	}

	if callCount != attempts {
		t.Errorf("期望函数被调用%d次，但实际调用了%d次", attempts, callCount)
	}
}

func TestRetryWithBackoff_MaxDelayLimit(t *testing.T) {
	attempts := 3
	initialDelay := 100 * time.Millisecond
	maxDelay := 50 * time.Millisecond // maxDelay 小于 initialDelay

	callCount := 0
	startTime := time.Now()

	fn := func() error {
		callCount++
		return errors.New("错误")
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err == nil {
		t.Error("期望失败，但没有返回错误")
	}

	elapsed := time.Since(startTime)

	// 验证延迟被限制在 maxDelay 内
	expectedMinDelay := maxDelay + maxDelay // 两次重试都应该使用 maxDelay
	if elapsed < expectedMinDelay {
		t.Errorf("期望至少延迟 %v，但实际延迟了 %v", expectedMinDelay, elapsed)
	}

	if callCount != attempts {
		t.Errorf("期望函数被调用%d次，但实际调用了%d次", attempts, callCount)
	}
}

func TestRetryWithBackoff_Jitter(t *testing.T) {
	attempts := 3
	initialDelay := 20 * time.Millisecond
	maxDelay := 100 * time.Millisecond

	callCount := 0
	startTime := time.Now()

	fn := func() error {
		callCount++
		return errors.New("错误")
	}

	err := RetryWithBackoff(attempts, initialDelay, maxDelay, fn)

	if err == nil {
		t.Error("期望失败，但没有返回错误")
	}

	elapsed := time.Since(startTime)

	// 验证延迟包含抖动，应该大于基础延迟
	expectedMinDelay := initialDelay + initialDelay*2
	if elapsed < expectedMinDelay {
		t.Errorf("期望至少延迟 %v，但实际延迟了 %v", expectedMinDelay, elapsed)
	}

	if callCount != attempts {
		t.Errorf("期望函数被调用%d次，但实际调用了%d次", attempts, callCount)
	}
}

// 基准测试
func BenchmarkRetryWithBackoff_Success(b *testing.B) {
	attempts := 3
	initialDelay := 1 * time.Millisecond
	maxDelay := 10 * time.Millisecond

	fn := func() error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RetryWithBackoff(attempts, initialDelay, maxDelay, fn)
	}
}

func BenchmarkRetryWithBackoff_Failure(b *testing.B) {
	attempts := 3
	initialDelay := 1 * time.Millisecond
	maxDelay := 10 * time.Millisecond

	fn := func() error {
		return errors.New("错误")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RetryWithBackoff(attempts, initialDelay, maxDelay, fn)
	}
}
