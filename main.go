package main

import (
	"fmt"
	"os"
	fp "path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/quick"
	"github.com/therecipe/qt/widgets"
)

// CustomQuickView is view for displaying QML
type CustomQuickView struct {
	quick.QQuickView
	_ func() `slot:"reload,auto"`
}

func (v *CustomQuickView) reload() {
	v.Engine().ClearComponentCache()
	v.SetSource(v.Source())
	logrus.Println(v.Source().ToLocalFile())
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "liveqml [path]",
		Short: "Run live QML viewer for specified path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get absolute directory path from args
			path := args[0]
			dirPath := fp.Dir(path)
			dirPath, err := fp.Abs(dirPath)
			if err != nil {
				logrus.Errorln("Failed to get directory path:", err)
				return
			}

			// Create watcher for that dir
			watcher, err := createWatcher(dirPath)
			if err != nil {
				logrus.Errorln("Failed to create watcher:", err)
				return
			}
			defer watcher.Close()

			// Create app
			core.QCoreApplication_SetAttribute(core.Qt__AA_EnableHighDpiScaling, true)
			app := widgets.NewQApplication(len(os.Args), os.Args)
			fmt.Println(dirPath)

			// Create viewer
			view := NewCustomQuickView(nil)
			view.SetTitle("Live " + path)
			view.SetResizeMode(quick.QQuickView__SizeRootObjectToView)
			view.SetSource(core.QUrl_FromLocalFile(path))
			view.ShowMaximized()

			// If files in dir changed, update view
			go func() {
				for {
					select {
					case event := <-watcher.Events:
						fName := fp.Base(event.Name)
						if fp.Ext(fName) == ".qml" || strings.Contains(fName, ".qmlc.") {
							continue
						}

						logrus.Println("Event:", fp.Ext(fName), event)
						view.Reload()
					case err := <-watcher.Errors:
						logrus.Errorln("Watcher error:", err)
					}
				}
			}()

			// Exec app
			app.Exec()
		},
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalln(err)
	}
}

func createWatcher(dir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fp.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	return watcher, nil
}
