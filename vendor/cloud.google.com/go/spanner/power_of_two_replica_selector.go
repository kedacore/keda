/*
Copyright 2026 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spanner

import (
	"math"
	"math/rand"
)

type powerOfTwoReplicaSelector struct {
	intn func(int) int
}

var defaultPowerOfTwoReplicaSelector = newPowerOfTwoReplicaSelector()

func newPowerOfTwoReplicaSelector() powerOfTwoReplicaSelector {
	return powerOfTwoReplicaSelector{intn: rand.Intn}
}

func (s powerOfTwoReplicaSelector) chooseIndex(candidateCount int, scoreLookup func(int) float64) int {
	if candidateCount == 0 {
		return -1
	}
	if candidateCount == 1 {
		return 0
	}

	index1 := s.intn(candidateCount)
	index2 := s.intn(candidateCount - 1)
	if index2 >= index1 {
		index2++
	}

	score1 := scoreLookup(index1)
	score2 := scoreLookup(index2)
	if math.IsNaN(score1) {
		score1 = math.MaxFloat64
	}
	if math.IsNaN(score2) {
		score2 = math.MaxFloat64
	}
	if score1 <= score2 {
		return index1
	}
	return index2
}
