//go:build go1.18
// +build go1.18

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package generated

type TransactionalContentSetter interface {
	SetCRC64([]byte)
}

func (a *PathClientAppendDataOptions) SetCRC64(v []byte) {
	a.TransactionalContentCRC64 = v
}
