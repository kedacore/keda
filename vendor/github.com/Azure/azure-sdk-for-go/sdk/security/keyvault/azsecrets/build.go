//go:generate tsp-client update
//go:generate go run ./internal/transforms.go
//go:generate rm ./constants.go
//go:generate goimports -w .

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.

package azsecrets
