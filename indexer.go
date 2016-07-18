package indexer

import "sync"

const (
	// OK is returned to the user when the requested operation succeeded.
	OK = "OK\n"

	// Fail is returned to the user when the requested operation cannot be completed due to some depedencies constraints violation.
	Fail = "FAIL\n"

	// Error is returned to the user when the user sent an unknown command or the  message is malformed.
	Error = "ERROR\n"
)

// Indexer keeps track of packages and their dependencies.
type Indexer interface {
	Index(*Pkg) string
	Remove(string) string
	Query(string) string
}

// InMemoryIndexer holds an in-memory registry.
type InMemoryIndexer struct {
	registry map[string]*Pkg
	m        *sync.Mutex
}

// NewInMemoryIndexer returns a new InMemoryIndexer instance.
func NewInMemoryIndexer() *InMemoryIndexer {
	return &InMemoryIndexer{
		registry: map[string]*Pkg{},
		m:        &sync.Mutex{},
	}
}

// Index adds p and its dependencies to registry.
// It returns OK if p could be indexed or if it was already present.
// It returns Fail if p cannot be indexed because some of its dependencies aren't indexed yet and need to be installed first.
func (i *InMemoryIndexer) Index(p *Pkg) string {
	i.m.Lock()
	defer i.m.Unlock()

	if _, exist := i.registry[p.Name]; exist {
		return OK
	}

	if !i.canIndex(p) {
		return Fail
	}

	i.registry[p.Name] = p
	return OK
}

// Remove removes package name from i.
// It returns OK if name could be removed from the index, or if name wasn't indexed.
// It returns Fail if name could not be removed from the index because some other indexed package depends on it.
func (i *InMemoryIndexer) Remove(name string) string {
	i.m.Lock()
	defer i.m.Unlock()

	if _, exist := i.registry[name]; !exist {
		return OK
	}

	if !i.canRemove(name) {
		return Fail
	}

	delete(i.registry, name)
	return OK
}

// Query checks if name is indexed in i.
// It returns OK if the package is indexed.
// It returns Fail if the package isn't indexed.
func (i *InMemoryIndexer) Query(name string) string {
	if _, exist := i.registry[name]; exist {
		return OK
	}

	return Fail
}

func (i *InMemoryIndexer) count() int {
	return len(i.registry)
}

func (i *InMemoryIndexer) canIndex(p *Pkg) bool {
	for _, d := range p.Deps {
		if _, exist := i.registry[d]; !exist {
			return false
		}
	}

	return true
}

func (i *InMemoryIndexer) canRemove(name string) bool {
	ok := true
	for _, p := range i.registry {
		for _, dep := range p.Deps {
			if dep == name {
				ok = false
				break
			}
		}
	}
	return ok
}
