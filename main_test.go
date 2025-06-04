package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func s(s ...string) []string {
	return s
}

func TestPartitionFiles(t *testing.T) {
	run := func(name string, all, markers []string, expected [][]string) {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, partitionFiles(all, markers))
		})
	}

	run("one", s("one"), s("one"), [][]string{{"one"}})
	run(
		"happy path",
		s("one", "two", "three"),
		s("two"),
		[][]string{{"one"}, {"two", "three"}},
	)
	run(
		"file per dir",
		s("one", "two", "three"),
		s("one", "two", "three"),
		[][]string{{"one"}, {"two"}, {"three"}},
	)
	run("first", s("one", "two"), s("one"), [][]string{{"one", "two"}})
	run("last", s("one", "two"), s("two"), [][]string{{"one"}, {"two"}})
}
