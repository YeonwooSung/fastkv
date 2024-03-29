package bloomfilter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMutex(t *testing.T) {
	t.Parallel()

	exclusiveMutex, err := NewMutex(LockTypeExclusive)
	assert.NoError(t, err)
	assert.IsType(t, (*ExclusiveMutex)(nil), exclusiveMutex)

	readWriteMutex, err := NewMutex(LockTypeReadWrite)
	assert.NoError(t, err)
	assert.IsType(t, (*ReadWriteMutex)(nil), readWriteMutex)

	_, err = NewMutex(0)
	assert.Error(t, err)
}

func TestMutexLocks(t *testing.T) {
	type testCase struct {
		name         string
		mutex        Mutex
		lock         bool
		getLockFuncs func(mutex Mutex) (func(), func(), func())
	}

	tests := []testCase{
		{
			name:  "ExclusiveMutex_Read_Read",
			mutex: &ExclusiveMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.RLock, mutex.RLock, mutex.RUnlock
			},
			lock: true,
		},
		{
			name:  "ExclusiveMutex_Write_Read",
			mutex: &ExclusiveMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.WLock, mutex.RLock, mutex.RUnlock
			},
			lock: true,
		},
		{
			name:  "ExclusiveMutex_Read_Write",
			mutex: &ExclusiveMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.WLock, mutex.RLock, mutex.RUnlock
			},
			lock: true,
		},
		{
			name:  "ReadWriteMutex_Write_Write",
			mutex: &ExclusiveMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.WLock, mutex.WLock, mutex.WUnlock
			},
			lock: true,
		},
		{
			name:  "ReadWriteMutex_Read_Read",
			mutex: &ReadWriteMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.RLock, mutex.RLock, mutex.RUnlock
			},
			lock: false,
		},
		{
			name:  "ReadWriteMutex_Write_Read",
			mutex: &ReadWriteMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.WLock, mutex.RLock, mutex.RUnlock
			},
			lock: true,
		},
		{
			name:  "ReadWriteMutex_Read_Write",
			mutex: &ReadWriteMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.WLock, mutex.RLock, mutex.RUnlock
			},
			lock: true,
		},
		{
			name:  "ExclusiveMutex_Write_Write",
			mutex: &ReadWriteMutex{},
			getLockFuncs: func(mutex Mutex) (func(), func(), func()) {
				return mutex.WLock, mutex.WLock, mutex.WUnlock
			},
			lock: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			done := make(chan bool)
			timeout := time.After(200 * time.Millisecond)
			lock1, lock2, unlock2 := tc.getLockFuncs(tc.mutex)

			lock1()

			go func() {
				lock2()
				defer unlock2()
				done <- true
			}()

			select {
			case <-done:
				assert.Equal(t, false, tc.lock, "Test passed, but should have locked")
				return
			case <-timeout:
				assert.Equal(t, true, tc.lock, "Test got locked, but should have passed")
				return
			}
		})
	}
}
