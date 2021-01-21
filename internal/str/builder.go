package str

import (
	"strings"
	"sync"
)

// Builder is an hacky goroutine safe strings.Builder
// to use when testing
type Builder struct {
	mutex   sync.Mutex
	builder strings.Builder
}

func (b *Builder) Write(p []byte) (n int, err error) {
	b.mutex.Lock()
	n, err = b.builder.Write(p)
	b.mutex.Unlock()

	return n, err
}

// Reset the internal buffer state
func (b *Builder) Reset() {
	b.mutex.Lock()
	b.builder.Reset()
	b.mutex.Unlock()
}

func (b *Builder) String() string {
	b.mutex.Lock()
	val := b.builder.String()
	b.mutex.Unlock()

	return val
}
