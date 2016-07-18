package indexer

import (
	"sync"
	"testing"
)

func TestIndex_Fail_MissingDeps(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	pkg := &Pkg{
		Name: "nginx",
		Deps: []string{"pcre-8.38", "zlib-1.2.8"},
	}

	if res := fixture.Index(pkg); res != Fail {
		t.Errorf("Expected Index to return %q, but got %q", Fail, res)
	}

	// index first dep. Expect to Fail.
	seedRegistry(fixture, &Pkg{Name: "pcre-8.38"})
	if res := fixture.Index(pkg); res != Fail {
		t.Errorf("Expected Index to return %q, but got %q", Fail, res)
	}

	// index second dep. Expect to pass.
	seedRegistry(fixture, &Pkg{Name: "zlib-1.2.8"})
	if res := fixture.Index(pkg); res != OK {
		t.Errorf("Expected Index to return %q, but got %q", OK, res)
	}

	assertExist(fixture, pkg, t)
}

func TestIndex_OK_NewPkg(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	pkg := &Pkg{
		Name: "mysql",
		Deps: []string{"mysql-client-core-5.5"},
	}
	dependency := &Pkg{Name: "mysql-client-core-5.5"}
	seedRegistry(fixture, dependency)

	if res := fixture.Index(pkg); res != OK {
		t.Errorf("Expected Index to return %q, but got %q", OK, res)
	}

	assertExist(fixture, pkg, t)
}

func TestIndex_OK_ExistingPkg(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	pkg := &Pkg{
		Name: "mysql",
		Deps: []string{"mysql-client-core-5.5"},
	}
	dependency := &Pkg{Name: "mysql-client-core-5.5"}
	seedRegistry(fixture, pkg, dependency)

	if res := fixture.Index(pkg); res != OK {
		t.Errorf("Expected Index to return %q, but got %q", OK, res)
	}
}

func TestIndex_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	mysql := &Pkg{Name: "mysql", Deps: []string{"mysql-client-core-5.5"}}
	nginx := &Pkg{Name: "nginx", Deps: []string{"pcre-8.38", "zlib-1.2.8"}}
	haproxy := &Pkg{Name: "haproxy", Deps: []string{"libpcre3-dev", "build-essential-11.5"}}
	pkgs := []*Pkg{mysql, nginx, haproxy, mysql} // repeat mysql
	allDeps := []*Pkg{
		&Pkg{Name: "mysql-client-core-5.5"},
		&Pkg{Name: "pcre-8.38"},
		&Pkg{Name: "zlib-1.2.8"},
		&Pkg{Name: "libpcre3-dev"},
		&Pkg{Name: "build-essential-11.5"},
	}
	seedRegistry(fixture, allDeps...)

	res := make(chan string, len(pkgs))
	w := &sync.WaitGroup{}
	w.Add(len(pkgs))
	for _, f := range pkgs {
		go func(p *Pkg) {
			res <- fixture.Index(p)
			w.Done()
		}(f)
	}
	w.Wait()
	close(res)

	if fixture.count() != 8 { // pkgs + dependencies = 8
		t.Errorf("Expected registry to have %d packages, but got %d", len(pkgs), fixture.count())
	}

	if !t.Failed() {
		for r := range res {
			if r != OK {
				t.Error("Expected all responses to be OK")
			}
		}
	}
}

func TestRemove_OK_NotExist(t *testing.T) {
	fixture := NewInMemoryIndexer()
	pkg := &Pkg{Name: "mysql"}

	assertNotExist(fixture, pkg, t)
	if res := fixture.Remove(pkg.Name); res != OK {
		t.Errorf("Expected Remove() to return %q, but got %q", OK, res)
	}
}

func TestRemove_OK_Exist(t *testing.T) {
	fixture := NewInMemoryIndexer()
	pkg := &Pkg{Name: "mysql"}
	seedRegistry(fixture, pkg)

	if res := fixture.Remove(pkg.Name); res != OK {
		t.Errorf("Expected Remove() to return %q, but got %q", OK, res)
	}

	if !t.Failed() {
		assertNotExist(fixture, pkg, t)
	}
}

func TestRemove_Fail_HasDependents(t *testing.T) {
	fixture := NewInMemoryIndexer()
	dependency := &Pkg{Name: "mysql-client-core-5.5"}
	pkg := &Pkg{Name: "mysql", Deps: []string{dependency.Name}}
	seedRegistry(fixture, dependency, pkg)

	if res := fixture.Remove(dependency.Name); res != Fail {
		t.Errorf("Expected Remove() to return %q, but got %q", Fail, res)
	}

	// expect dependency is not removed
	assertExist(fixture, dependency, t)
}

func TestRemove_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	mysql, nginx, httpd := &Pkg{Name: "mysql"}, &Pkg{Name: "nginx"}, &Pkg{Name: "httpd"}
	pkgs := []*Pkg{mysql, nginx, httpd, mysql} // repeat mysql
	seedRegistry(fixture, pkgs...)

	res := make(chan string, len(pkgs))
	w := &sync.WaitGroup{}
	w.Add(len(pkgs))
	for _, pkg := range pkgs {
		go func(p *Pkg) {
			res <- fixture.Remove(p.Name)
			w.Done()
		}(pkg)
	}
	w.Wait()
	close(res)

	for r := range res {
		if r != OK {
			t.Errorf("Expected response to be %q, but got %q", OK, r)
		}
	}

	for _, pkg := range pkgs {
		assertNotExist(fixture, pkg, t)
	}
}

func TestQuery_Fail_NotExist(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	pkg := &Pkg{Name: "mysql"}

	if res := fixture.Query(pkg.Name); res != Fail {
		t.Errorf("Expected Query() to return %q, but got %q", Fail, res)
	}
}

func TestQuery_OK_Exist(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()
	pkg := &Pkg{Name: "mysql"}
	seedRegistry(fixture, pkg)

	if res := fixture.Query(pkg.Name); res != OK {
		t.Errorf("Expected Query() to return %q, but got %q", OK, res)
	}
}

func TestQuery_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	fixture := NewInMemoryIndexer()

	mysql, nginx, haproxy := &Pkg{Name: "mysql"}, &Pkg{Name: "nginx"}, &Pkg{Name: "haproxy"}
	pkgs := []*Pkg{mysql, nginx, haproxy}
	seedRegistry(fixture, pkgs...)

	res := make(chan string, len(pkgs))
	w := &sync.WaitGroup{}
	w.Add(len(pkgs))
	for _, pkg := range pkgs {
		go func(p *Pkg) {
			res <- fixture.Query(p.Name)
			w.Done()
		}(pkg)
	}

	w.Wait()
	close(res)

	for r := range res {
		if r != OK {
			t.Errorf("Expected Query() to return %q, but got %q", OK, r)
		}
	}
}

// assertExist asserts that pkg are indexed in i.
// It also compares the dependencies of pkg with that returned by i.
func assertExist(i *InMemoryIndexer, pkg *Pkg, t *testing.T) {
	p, exist := i.registry[pkg.Name]
	if !exist {
		t.Errorf("Expected package %q to be indexed", pkg.Name)
	}

	if len(p.Deps) != len(pkg.Deps) {
		t.Errorf("Expected dependencies count to be %d, but got %d", len(p.Deps), len(pkg.Deps))
	}

	if !t.Failed() {
		for i, d := range pkg.Deps {
			if p.Deps[i] != d {
				t.Errorf("Expected dependencies of %q to be %q, but got %q", pkg.Name, d, p.Deps[i])
			}
		}
	}
}

// assertNotExist asserts that pkg are not indexed in i.
func assertNotExist(i *InMemoryIndexer, pkg *Pkg, t *testing.T) {
	if _, exist := i.registry[pkg.Name]; exist {
		t.Errorf("Expected package %q to be removed", pkg.Name)
	}
}

// seedRegistry is a helper function to help add pkgs to i.
func seedRegistry(i *InMemoryIndexer, pkgs ...*Pkg) {
	for _, p := range pkgs {
		i.registry[p.Name] = p
	}
}
