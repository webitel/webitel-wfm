package resolver

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc/resolver"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const name = "discovery"

var ErrWatcherCreateTimeout = werror.New("discovery create watcher overtime")

// Option is builder option.
type Option func(o *builder)

// WithTimeout with timeout option.
func WithTimeout(timeout time.Duration) Option {
	return func(b *builder) {
		b.timeout = timeout
	}
}

// WithInsecure with isSecure option.
func WithInsecure(insecure bool) Option {
	return func(b *builder) {
		b.insecure = insecure
	}
}

// WithSubset with subset size.
func WithSubset(size int) Option {
	return func(b *builder) {
		b.subsetSize = size
	}
}

// PrintDebugLog print grpc resolver watch service log
func PrintDebugLog(p bool) Option {
	return func(b *builder) {
		b.debugLog = p
	}
}

type builder struct {
	log        *wlog.Logger
	discoverer registry.Discovery
	timeout    time.Duration
	insecure   bool
	subsetSize int
	debugLog   bool
}

// NewBuilder creates a builder which is used to factory registry resolvers.
func NewBuilder(log *wlog.Logger, d registry.Discovery, opts ...Option) resolver.Builder {
	b := &builder{
		log:        log,
		discoverer: d,
		timeout:    time.Second * 3,
		insecure:   false,
		debugLog:   true,
		subsetSize: 25,
	}

	for _, o := range opts {
		o(b)
	}

	return b
}

func (b *builder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	watchRes := &struct {
		err error
		w   registry.Watcher
	}{}

	done := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		w, err := b.discoverer.Watch(ctx, strings.TrimPrefix(target.URL.Path, "/"))
		watchRes.w = w
		watchRes.err = err
		close(done)
	}()

	var err error
	if b.timeout > 0 {
		select {
		case <-done:
			err = watchRes.err
		case <-time.After(b.timeout):
			err = ErrWatcherCreateTimeout
		}
	} else {
		<-done
		err = watchRes.err
	}

	if err != nil {
		cancel()

		return nil, err
	}

	r := &discoveryResolver{
		log:         b.log,
		w:           watchRes.w,
		cc:          cc,
		ctx:         ctx,
		cancel:      cancel,
		insecure:    b.insecure,
		debugLog:    b.debugLog,
		subsetSize:  b.subsetSize,
		selectorKey: uuid.New().String(),
	}

	go r.watch()

	return r, nil
}

// Scheme return scheme of discovery
func (*builder) Scheme() string {
	return name
}