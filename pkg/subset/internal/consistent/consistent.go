// Copyright (C) 2012 Numerotron Inc.
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

// Package consistent provides a consistent hashing function.
//
// Consistent hashing is often used to distribute requests to a changing set of servers.  For example,
// say you have some cache servers cacheA, cacheB, and cacheC.  You want to decide which cache server
// to use to look up information on a user.
//
// You could use a typical hash table and hash the user id
// to one of cacheA, cacheB, or cacheC.  But with a typical hash table, if you add or remove a server,
// almost all keys will get remapped to different results, which basically could bring your service
// to a grinding halt while the caches get rebuilt.
//
// With a consistent hash, adding or removing a server drastically reduces the number of keys that
// get remapped.
//
// Read more about consistent hashing on wikipedia:  http://en.wikipedia.org/wiki/Consistent_hashing
package consistent // import "stathat.com/c/consistent"

import (
	"errors"
	"hash/crc32"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
)

type Member interface {
	String() string
}

type uints []uint32

// Len returns the length of the uints array.
func (x uints) Len() int { return len(x) }

// Less returns true if element i is less than element j.
func (x uints) Less(i, j int) bool { return x[i] < x[j] }

// Swap exchanges elements i and j.
func (x uints) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// ErrEmptyCircle is the error returned when trying to get an element when nothing has been added to hash.
var ErrEmptyCircle = errors.New("empty circle")

// Consistent holds the information about the members of the consistent hash circle.
type Consistent[M Member] struct {
	circle           map[uint32]M
	members          map[string]bool
	sortedHashes     uints
	NumberOfReplicas int
	count            int64
	scratch          [64]byte //nolint:unused
	UseFnv           bool
	sync.RWMutex
}

// New creates a new Consistent object with a default setting of 20 replicas for each entry.
//
// To change the number of replicas, set NumberOfReplicas before adding entries.
func New[M Member]() *Consistent[M] {
	c := new(Consistent[M])
	c.NumberOfReplicas = 20
	c.circle = make(map[uint32]M)
	c.members = make(map[string]bool)
	return c
}

// eltKey generates a string key for an element with an index.
func (c *Consistent[M]) eltKey(elt string, idx int) string {
	// return elt + "|" + strconv.Itoa(idx)
	return strconv.Itoa(idx) + elt
}

// Add inserts a string element in the consistent hash.
func (c *Consistent[M]) Add(elt M) {
	c.Lock()
	defer c.Unlock()
	c.add(elt)
}

// need c.Lock() before calling
func (c *Consistent[M]) add(elt M) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		c.circle[c.hashKey(c.eltKey(elt.String(), i))] = elt
	}
	c.members[elt.String()] = true
	c.updateSortedHashes()
	c.count++
}

// Remove removes an element from the hash.
func (c *Consistent[M]) Remove(elt M) {
	c.Lock()
	defer c.Unlock()
	c.remove(elt.String())
}

// need c.Lock() before calling
func (c *Consistent[M]) remove(elt string) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		delete(c.circle, c.hashKey(c.eltKey(elt, i)))
	}
	delete(c.members, elt)
	c.updateSortedHashes()
	c.count--
}

// Set sets all the elements in the hash.  If there are existing elements not
// present in elts, they will be removed.
func (c *Consistent[M]) Set(elts []M) {
	c.Lock()
	defer c.Unlock()
	for k := range c.members {
		found := false
		for _, v := range elts {
			if k == v.String() {
				found = true
				break
			}
		}
		if !found {
			c.remove(k)
		}
	}
	for _, v := range elts {
		_, exists := c.members[v.String()]
		if exists {
			continue
		}
		c.add(v)
	}
}

func (c *Consistent[M]) Members() []string {
	c.RLock()
	defer c.RUnlock()
	var m []string
	for k := range c.members {
		m = append(m, k)
	}
	return m
}

// Get returns an element close to where name hashes to in the circle.
func (c *Consistent[M]) Get(name string) (res M, err error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		err = ErrEmptyCircle
		return
	}
	key := c.hashKey(name)
	i := c.search(key)
	res = c.circle[c.sortedHashes[i]]
	return
}

func (c *Consistent[M]) search(key uint32) (i int) {
	f := func(x int) bool {
		return c.sortedHashes[x] > key
	}
	i = sort.Search(len(c.sortedHashes), f)
	if i >= len(c.sortedHashes) {
		i = 0
	}
	return
}

// GetTwo returns the two closest distinct elements to the name input in the circle.
func (c *Consistent[M]) GetTwo(name string) (a M, b M, err error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		err = ErrEmptyCircle
		return
	}
	key := c.hashKey(name)
	i := c.search(key)
	a = c.circle[c.sortedHashes[i]]

	if c.count == 1 {
		return
	}

	start := i
	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		b = c.circle[c.sortedHashes[i]]
		if b.String() != a.String() {
			break
		}
	}
	return a, b, nil
}

// GetN returns the N closest distinct elements to the name input in the circle.
func (c *Consistent[M]) GetN(name string, n int) (res []M, err error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.circle) == 0 {
		err = ErrEmptyCircle
		return
	}

	if c.count < int64(n) {
		n = int(c.count)
	}

	var (
		key   = c.hashKey(name)
		i     = c.search(key)
		start = i
		elem  = c.circle[c.sortedHashes[i]]
	)
	res = make([]M, 0, n)
	res = append(res, elem)

	if len(res) == n {
		return res, nil
	}

	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		elem = c.circle[c.sortedHashes[i]]
		if !sliceContainsMember(res, elem) {
			res = append(res, elem)
		}
		if len(res) == n {
			break
		}
	}

	return res, nil
}

func (c *Consistent[M]) hashKey(key string) uint32 {
	if c.UseFnv {
		return c.hashKeyFnv(key)
	}
	return c.hashKeyCRC32(key)
}

func (c *Consistent[M]) hashKeyCRC32(key string) uint32 {
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		return crc32.ChecksumIEEE(scratch[:len(key)])
	}
	return crc32.ChecksumIEEE([]byte(key))
}

func (c *Consistent[M]) hashKeyFnv(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

func (c *Consistent[M]) updateSortedHashes() {
	hashes := c.sortedHashes[:0]
	// reallocate if we're holding on to too much (1/4th)
	if cap(c.sortedHashes)/(c.NumberOfReplicas*4) > len(c.circle) {
		hashes = nil
	}
	for k := range c.circle {
		hashes = append(hashes, k)
	}
	sort.Sort(hashes)
	c.sortedHashes = hashes
}

func sliceContainsMember[M Member](set []M, member M) bool {
	for _, m := range set {
		if m.String() == member.String() {
			return true
		}
	}
	return false
}
