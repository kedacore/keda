// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal // import "go.mongodb.org/mongo-driver/internal"

// Version is the current version of the driver.
var Version = "local build"

// LegacyHello is the legacy version of the hello command.
var LegacyHello = "isMaster"

// LegacyHelloLowercase is the lowercase, legacy version of the hello command.
var LegacyHelloLowercase = "ismaster"

// LegacyNotPrimary is the legacy version of the "not primary" server error message.
var LegacyNotPrimary = "not master"
