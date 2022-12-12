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

package collection

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](values []T) Set[T] {
	result := make(Set[T], len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func (set Set[T]) Union(other Set[T]) {
	set.AddAll(other.Values())
}

func (set Set[T]) AddAll(values []T) {
	for _, value := range values {
		set.Add(value)
	}
}

func (set Set[T]) Add(value T) {
	set[value] = struct{}{}
}

func (set Set[T]) RemoveAll(values []T) {
	for _, value := range values {
		set.Remove(value)
	}
}

func (set Set[T]) Remove(value T) {
	delete(set, value)
}

func (set Set[T]) Values() []T {
	if len(set) == 0 {
		return nil
	}
	result := make([]T, len(set))
	i := 0
	for value := range set {
		result[i] = value
		i++
	}
	return result
}

func (set Set[T]) Copy() Set[T] {
	result := make(map[T]struct{}, len(set))
	for k, v := range set {
		result[k] = v
	}
	return result
}
