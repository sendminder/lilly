//go:build tools
// +build tools

// This file imports packages that are used when running go generate, or used
// during the development process but not otherwise depended on by built code.
//
// @see https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

import (
	_ "github.com/daixiang0/gci"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "github.com/rakyll/gotest"
	_ "go.uber.org/mock/mockgen"
	_ "golang.org/x/tools/cmd/goimports"
)
