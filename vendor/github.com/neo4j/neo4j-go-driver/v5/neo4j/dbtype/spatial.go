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

package dbtype

import (
	"fmt"
)

// Point2D represents a two dimensional point in a particular coordinate reference system.
type Point2D struct {
	X            float64
	Y            float64
	SpatialRefId uint32 // Id of coordinate reference system.
}

// Point3D represents a three dimensional point in a particular coordinate reference system.
type Point3D struct {
	X            float64
	Y            float64
	Z            float64
	SpatialRefId uint32 // Id of coordinate reference system.
}

// String returns string representation of this point.
func (p Point2D) String() string {
	return fmt.Sprintf("Point{srId=%d, x=%f, y=%f}", p.SpatialRefId, p.X, p.Y)
}

// String returns string representation of this point.
func (p Point3D) String() string {
	return fmt.Sprintf("Point{srId=%d, x=%f, y=%f, z=%f}", p.SpatialRefId, p.X, p.Y, p.Z)
}
