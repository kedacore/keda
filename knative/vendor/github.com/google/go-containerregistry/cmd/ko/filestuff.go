// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// FilenameOptions is from pkg/kubectl.
type FilenameOptions struct {
	Filenames []string
	Recursive bool
	Watch     bool
}

func addFileArg(cmd *cobra.Command, fo *FilenameOptions) {
	// From pkg/kubectl
	cmd.Flags().StringSliceVarP(&fo.Filenames, "filename", "f", fo.Filenames,
		"Filename, directory, or URL to files to use to create the resource")
	cmd.Flags().BoolVarP(&fo.Recursive, "recursive", "R", fo.Recursive,
		"Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
	cmd.Flags().BoolVarP(&fo.Watch, "watch", "W", fo.Watch,
		"Continuously monitor the transitive dependencies of the passed yaml files, and redeploy whenever anything changes.")
}

// Based heavily on pkg/kubectl
func enumerateFiles(fo *FilenameOptions) chan string {
	files := make(chan string)
	go func() {
		// When we're done enumerating files, close the channel
		defer close(files)
		// When we are in --watch mode, we set up watches on the filesystem locations
		// that we are supplied and continuously stream files, until we are sent an
		// interrupt.
		var watcher *fsnotify.Watcher
		if fo.Watch {
			var err error
			watcher, err = fsnotify.NewWatcher()
			if err != nil {
				log.Fatalf("Unexpected error initializing fsnotify: %v", err)
			}
			defer watcher.Close()
		}
		for _, paths := range fo.Filenames {
			// Just pass through '-' as it is indicative of stdin.
			if paths == "-" {
				files <- paths
				continue
			}
			// For each of the "filenames" we are passed (file or directory) start a
			// "Walk" to enumerate all of the contained files recursively.
			err := filepath.Walk(paths, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// If this is a directory, skip it if it isn't the current directory we are
				// processing (unless we are in recursive mode).  If we decide to process
				// the directory, and we're in watch mode, then we set up a watch on the
				// directory.
				if fi.IsDir() {
					if path != paths && !fo.Recursive {
						return filepath.SkipDir
					}
					if watcher != nil {
						watcher.Add(path)
					}
					// We don't stream back directories, we just decide to skip them, or not.
					return nil
				}

				// Don't check extension if the filepath was passed explicitly
				if path != paths {
					switch filepath.Ext(path) {
					case ".json", ".yaml":
						// Process these.
					default:
						return nil
					}
					// We weren't passed this explicitly, so elide the watch as we
					// are already watching the directory.
				} else {
					// We were passed this directly, and so we may not be watching the
					// directory, so watch this file explicitly.
					if watcher != nil {
						watcher.Add(path)
					}
				}

				files <- path
				return nil
			})
			if err != nil {
				log.Fatalf("Error enumerating files: %v", err)
			}
		}

		// We're done watching the files we were passed and setting up watches.
		// Now listen for change events from the watches we set up and resend
		// files that change as if we just saw them (so they can be reprocessed).
		if watcher != nil {
			for {
				select {
				case event := <-watcher.Events:
					switch filepath.Ext(event.Name) {
					case ".json", ".yaml":
						files <- event.Name
					}
				case err := <-watcher.Errors:
					log.Fatalf("Error watching: %v", err)
				}
			}
		}
	}()
	return files
}
