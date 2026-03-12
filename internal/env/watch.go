package env

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches .env.<repo>.global and .env.<repo>.override files
// and re-runs Sync whenever any of them change.
type Watcher struct {
	watcher    *fsnotify.Watcher
	projectDir string
	repoNames  []string
	done       chan struct{}
}

// Watch starts watching env source files for changes and auto-syncs.
// Call Stop() to clean up.
func Watch(projectDir string, repoNames []string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	w := &Watcher{
		watcher:    fw,
		projectDir: projectDir,
		repoNames:  repoNames,
		done:       make(chan struct{}),
	}

	// Watch the directories containing the env files rather than
	// individual files, since editors often write via rename.
	dirs := make(map[string]struct{})
	dirs[EnvsDir()] = struct{}{}
	dirs[projectDir] = struct{}{}

	for dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fw.Close()
			return nil, err
		}
		if err := fw.Add(dir); err != nil {
			fw.Close()
			return nil, fmt.Errorf("watch %s: %w", dir, err)
		}
	}

	// Build set of filenames we care about for quick lookup
	relevant := make(map[string]struct{})
	for _, name := range repoNames {
		relevant[filepath.Base(GlobalPath(name))] = struct{}{}
		relevant[fmt.Sprintf(".env.%s.override", name)] = struct{}{}
	}

	go w.loop(relevant)

	return w, nil
}

func (w *Watcher) loop(relevant map[string]struct{}) {
	defer close(w.done)

	// Debounce: batch rapid changes into a single sync
	var timer *time.Timer
	debounce := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			base := filepath.Base(event.Name)
			if _, ok := relevant[base]; !ok {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, func() {
				Sync(w.projectDir, w.repoNames)
			})

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// Stop stops watching and cleans up resources.
func (w *Watcher) Stop() {
	w.watcher.Close()
	<-w.done
}
