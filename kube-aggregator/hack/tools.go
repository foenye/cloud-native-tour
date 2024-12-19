//go:build tools
// +build tools

// Package tools imports things required by build scripts, to force `go mod` to see them as dependencies
// go mod tidy; go mod vendor
package tools

import _ "k8s.io/code-generator"
