//go:build !1.13
// +build !1.13

package chttp

import (
	"golang.org/x/xerrors"
)

type printer = xerrors.Printer

var formatError = xerrors.FormatError
