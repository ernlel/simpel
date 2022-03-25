package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func NestedWatch(path string) (chan string, error) {

	var NestedWatchItems chan string

	var folders []string
	filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			folders = append(folders, newPath)
		}
		return nil
	})

	if len(folders) == 0 {
		return nil, errors.New("Nothing to watch.")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	NestedWatchItems = make(chan string)

	for _, folder := range folders {
		err := watcher.Add(folder)
		if err != nil {
			log.Println("Error watch: ", folder, err)
		}
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:

				if event.Op&fsnotify.Create == fsnotify.Create {
					NestedWatchItems <- event.Name
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					NestedWatchItems <- event.Name
				}

				if event.Op&fsnotify.Remove == fsnotify.Remove {
					NestedWatchItems <- event.Name
				}

			case err := <-watcher.Errors:
				log.Println("error", err)
			}
		}
	}()

	return NestedWatchItems, nil
}
