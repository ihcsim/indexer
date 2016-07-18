package indexer

import (
	"fmt"
	"strings"
)

const (
	msgSuffix          = "\n"
	msgDelimiter       = "|"
	msgDelimitersCount = 2
	depsDelimiter      = ","
	splitsMax          = 3

	// ErrMalformedMsg is an error message indicating a malformed message structure.
	ErrMalformedMsg = "Malformed message structure"

	// ErrMissingCmd is an error message indicating a command is missing.
	ErrMissingCmd = "Missing command"

	// ErrMissingName is an wrror message indicating a package name is missing.
	ErrMissingName = "Missing package name"
)

// ParseMsg extracts the package and command information from s.
func ParseMsg(s string) (p *Pkg, cmd string, e error) {
	if !isWellStructured(s) {
		return nil, "", fmt.Errorf(ErrMalformedMsg)
	}

	splits, err := extractFields(s)
	if err != nil {
		return nil, "", err
	}

	cmd = splits[0]
	p = &Pkg{Name: splits[1], Deps: extractDeps(splits)}
	return
}

func isWellStructured(s string) bool {
	if !strings.HasSuffix(s, msgSuffix) {
		return false
	}

	if counts := strings.Count(s, msgDelimiter); counts < msgDelimitersCount {
		return false
	}

	return true
}

func extractFields(s string) ([]string, error) {
	splits := strings.Split(strings.TrimSpace(s), msgDelimiter)
	if splits[0] == "" {
		return nil, fmt.Errorf(ErrMissingCmd)
	}

	if splits[1] == "" {
		return nil, fmt.Errorf(ErrMissingName)
	}

	return splits, nil
}

func extractDeps(splits []string) []string {
	var deps []string
	if len(splits) == splitsMax && splits[splitsMax-1] != "" {
		deps = strings.Split(splits[splitsMax-1], depsDelimiter)
	}
	return deps
}
