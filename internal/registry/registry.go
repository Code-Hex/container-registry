package registry

import "path/filepath"

// BasePath represents base path for this application.
var BasePath = "testdata"

// PathJoinWithBase joins any number of path elements with base path into a single path,
// separating them with an OS specific Separator.
func PathJoinWithBase(name string, p ...string) string {
	return filepath.Join(
		append(
			[]string{
				BasePath,
				name,
			},
			p...,
		)...,
	)
}
