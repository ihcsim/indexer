package indexer

// Pkg represents a package or library that can be installed in a system. It captures information of the package's dependencies.
type Pkg struct {
	Name string
	Deps []string
}
