package cache

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*Cache)
}

type optionFunc func(*Cache)

func (o optionFunc) apply(c *Cache) {
	o(c)
}
