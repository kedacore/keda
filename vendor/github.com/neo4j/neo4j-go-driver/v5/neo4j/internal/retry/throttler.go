/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package retry

import (
	"math/rand"
	"time"
)

type Throttler time.Duration

func (t Throttler) next() Throttler {
	delay := time.Duration(t)
	const delayJitter = 0.2
	jitter := float64(delay) * delayJitter
	return Throttler(2*delay - time.Duration(jitter) + time.Duration(2*jitter*rand.Float64()))
}

func (t Throttler) delay() time.Duration {
	return time.Duration(t)
}
