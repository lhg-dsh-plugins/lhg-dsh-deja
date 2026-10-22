// Package dejavu holds the embedded CHANGELOG so the binary can say what a
// new version brings on its first run after an upgrade (#4619). The changelog
// lives at the repository root, so this file sits beside it.
package dejavu

import _ "embed"

// Changelog is the full CHANGELOG.md, embedded at build time.
//
//go:embed CHANGELOG.md
var Changelog string
