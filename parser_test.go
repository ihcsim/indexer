package indexer

import "testing"

func TestParseMessage_WellFormedMsg(t *testing.T) {
	var tests = []struct {
		command      string
		name         string
		dependencies []string
		msg          string
		expected     *Pkg
	}{
		{command: "INDEX", name: "cloog", dependencies: []string{"gmp", "isl", "pkg-config"}, msg: "INDEX|cloog|gmp,isl,pkg-config\n", expected: nil},
		{command: "INDEX", name: "ceylon", msg: "INDEX|ceylon|\n", expected: nil},
		{command: "REMOVE", name: "cloog", msg: "REMOVE|cloog|\n", expected: nil},
		{command: "QUERY", name: "cloog", msg: "QUERY|cloog|\n", expected: nil},
	}

	for _, test := range tests {
		p, cmd, err := ParseMsg(test.msg)
		if err != nil {
			t.Fatal("Unexpected error: ", err)
		}

		if cmd != test.command {
			t.Errorf("Expected command to be %q, but got %q", test.command, cmd)
		}

		if p.Name != test.name {
			t.Errorf("Expected package name to be %q, but got %q", test.name, p.Name)
		}

		if len(p.Deps) != len(test.dependencies) {
			t.Errorf("Expected %s dependencies count to be %d, but got %d", p.Name, len(test.dependencies), len(p.Deps))
		}

		for i, d := range test.dependencies {
			if d != p.Deps[i] {
				t.Errorf("Expected %s to have %q as a dependency", p.Name, d)
			}
		}
	}
}

func TestParseMessage_BrokenMsg(t *testing.T) {
	var tests = []struct {
		msg    string
		reason string
	}{
		{msg: "INDEX|cloog|gmp,isl,pkg-config", reason: "Line break suffix is missing"},
		{msg: "|ceylon|\n", reason: "Command is missing"},
		{msg: "REMOVE||\n", reason: "Package name is missing"},
		{msg: "QUERY\n", reason: "Delimiters are missing"},
		{msg: "QUERY|ceylon\n", reason: "Delimiters are missing"},
	}

	for _, test := range tests {
		_, _, err := ParseMsg(test.msg)
		if err == nil {
			t.Fatal("Expected error didn't occur. Should fail because", test.reason)
		}
	}
}
