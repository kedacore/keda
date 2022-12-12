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

package neo4j

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/collection"
	"sync"
)

// Bookmarks is a holder for server-side bookmarks which are used for causally-chained sessions.
// See also CombineBookmarks.
// Note: this will be changed from being a type alias to being a struct in 6.0. Please use BookmarksFromRawValues for construction
// from raw values and BookmarksToRawValues for accessing the raw values.
type Bookmarks = []string

// BookmarkManager centralizes bookmark manager supply and notification
// This API is experimental and may be changed or removed without prior notice
type BookmarkManager interface {
	// UpdateBookmarks updates the bookmark tracked by this bookmark manager
	// previousBookmarks are the initial bookmarks of the bookmark holder (like a Session)
	// newBookmarks are the bookmarks that are received after completion of the bookmark holder operation (like the end of a Session)
	UpdateBookmarks(ctx context.Context, previousBookmarks, newBookmarks Bookmarks) error

	// GetBookmarks returns all the bookmarks tracked by this bookmark manager
	// Note: the order of the returned bookmark slice does not need to be deterministic
	GetBookmarks(ctx context.Context) (Bookmarks, error)
}

// BookmarkManagerConfig is an experimental API and may be changed or removed
// without prior notice
type BookmarkManagerConfig struct {
	// Initial bookmarks per database
	InitialBookmarks Bookmarks

	// Supplier providing external bookmarks
	BookmarkSupplier func(context.Context) (Bookmarks, error)

	// Hook called whenever bookmarks get updated
	// The hook is called with the database and the new bookmarks
	// Note: the order of the supplied bookmark slice is not guaranteed
	BookmarkConsumer func(ctx context.Context, bookmarks Bookmarks) error
}

type bookmarkManager struct {
	bookmarks        collection.Set[string]
	supplyBookmarks  func(context.Context) (Bookmarks, error)
	consumeBookmarks func(context.Context, Bookmarks) error
	mutex            sync.RWMutex
}

func (b *bookmarkManager) UpdateBookmarks(ctx context.Context, previousBookmarks, newBookmarks Bookmarks) error {
	if len(newBookmarks) == 0 {
		return nil
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	var bookmarksToNotify Bookmarks
	b.bookmarks.RemoveAll(previousBookmarks)
	b.bookmarks.AddAll(newBookmarks)
	bookmarksToNotify = b.bookmarks.Values()
	if b.consumeBookmarks != nil {
		return b.consumeBookmarks(ctx, bookmarksToNotify)
	}
	return nil
}

func (b *bookmarkManager) GetBookmarks(ctx context.Context) (Bookmarks, error) {
	var extraBookmarks Bookmarks
	if b.supplyBookmarks != nil {
		bookmarks, err := b.supplyBookmarks(ctx)
		if err != nil {
			return nil, err
		}
		extraBookmarks = bookmarks
	}
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if len(b.bookmarks) == 0 {
		return extraBookmarks, nil
	}
	bookmarks := b.bookmarks.Copy()
	if extraBookmarks == nil {
		return bookmarks.Values(), nil
	}
	bookmarks.AddAll(extraBookmarks)
	return bookmarks.Values(), nil
}

func NewBookmarkManager(config BookmarkManagerConfig) BookmarkManager {
	return &bookmarkManager{
		bookmarks:        collection.NewSet(config.InitialBookmarks),
		supplyBookmarks:  config.BookmarkSupplier,
		consumeBookmarks: config.BookmarkConsumer,
	}
}

// CombineBookmarks is a helper method to combine []Bookmarks into a single Bookmarks instance.
// Let s1, s2, s3 be Session interfaces. You can easily causally chain the sessions like so:
// ```go
//
//	s4 := driver.NewSession(neo4j.SessionConfig{
//		Bookmarks: neo4j.CombineBookmarks(s1.LastBookmarks(), s2.LastBookmarks(), s3.LastBookmarks()),
//	})
//
// ```
// The server will then make sure to execute all transactions in s4 after any that were already executed in s1, s2, or s3
// at the time of calling LastBookmarks.
func CombineBookmarks(bookmarks ...Bookmarks) Bookmarks {
	var lenSum int
	for _, b := range bookmarks {
		lenSum += len(b)
	}
	res := make([]string, lenSum)
	var i int
	for _, b := range bookmarks {
		i += copy(res[i:], b)
	}
	return res
}

// BookmarksToRawValues exposes the raw server-side bookmarks.
// You should not need to use this method unless you want to serialize bookmarks.
// See Session.LastBookmarks and CombineBookmarks for alternatives.
func BookmarksToRawValues(bookmarks Bookmarks) []string {
	return bookmarks
}

// BookmarksFromRawValues creates Bookmarks from raw server-side bookmarks.
// You should not need to use this method unless you want to de-serialize bookmarks.
// See Session.LastBookmarks and CombineBookmarks for alternatives.
func BookmarksFromRawValues(values ...string) Bookmarks {
	return values
}
