package os

import (
	stdos "os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"

	. "github.com/candid82/joker/core"
)

type fileWatcher struct {
	watcher *fsnotify.Watcher
	ch      *Channel

	recursive bool
	done      chan struct{}

	closeOnce  sync.Once
	cancelOnce sync.Once
}

func watch(paths Seqable, ch *Channel, opts Map) Object {
	recursive := false
	if ok, obj := opts.Get(MakeKeyword("recursive?")); ok {
		recursive = EnsureObjectIsBoolean(obj, "recursive?: %s").B
	}

	watcher, err := fsnotify.NewWatcher()
	PanicOnErr(err)

	fw := &fileWatcher{
		watcher:   watcher,
		ch:        ch,
		recursive: recursive,
		done:      make(chan struct{}),
	}

	if err := fw.addPaths(paths); err != nil {
		fw.closeWatcher()
		PanicOnErr(err)
	}

	go fw.run()

	return Proc{
		Fn: func(args []Object) Object {
			CheckArity(args, 0, 0)
			fw.cancel()
			return NIL
		},
		Name:    "watch-cancel",
		Package: "std/os",
	}
}

func (fw *fileWatcher) cancel() {
	RT.GIL.Unlock()
	defer RT.GIL.Lock()

	fw.cancelOnce.Do(func() {
		fw.closeWatcher()
		fw.ch.Close()
		<-fw.done
	})
}

func (fw *fileWatcher) closeWatcher() {
	fw.closeOnce.Do(func() {
		_ = fw.watcher.Close()
	})
}

func (fw *fileWatcher) addPaths(paths Seqable) error {
	for s := paths.Seq(); !s.IsEmpty(); s = s.Rest() {
		path := EnsureObjectIsString(s.First(), "watch path: %s").S
		if err := fw.addPath(path); err != nil {
			return err
		}
	}
	return nil
}

func (fw *fileWatcher) addPath(path string) error {
	if !fw.recursive {
		return fw.watcher.Add(path)
	}

	info, err := stdos.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fw.watcher.Add(path)
	}
	return filepath.WalkDir(path, func(path string, d stdos.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return fw.watcher.Add(path)
		}
		return nil
	})
}

func (fw *fileWatcher) run() {
	defer close(fw.done)

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			if fw.recursive && event.Op&fsnotify.Create != 0 {
				fw.addCreatedDir(event.Name)
			}
			if !fw.send(watchEvent(event)) {
				fw.closeWatcher()
				return
			}
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			if !fw.send(watchError(err)) {
				fw.closeWatcher()
				return
			}
		}
	}
}

func (fw *fileWatcher) addCreatedDir(path string) {
	info, err := stdos.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	if err := filepath.WalkDir(path, func(path string, d stdos.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return fw.watcher.Add(path)
		}
		return nil
	}); err != nil {
		fw.send(watchError(err))
	}
}

func (fw *fileWatcher) send(obj Object) bool {
	return fw.ch.Send(obj)
}

func watchEvent(event fsnotify.Event) Object {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	m := EmptyArrayMap()
	m.Add(MakeKeyword("type"), MakeKeyword("event"))
	m.Add(MakeKeyword("path"), MakeString(event.Name))
	m.Add(MakeKeyword("ops"), watchOps(event.Op))
	return m
}

func watchError(err error) Object {
	RT.GIL.Lock()
	defer RT.GIL.Unlock()

	m := EmptyArrayMap()
	m.Add(MakeKeyword("type"), MakeKeyword("error"))
	m.Add(MakeKeyword("error"), RT.NewError(err.Error()))
	return m
}

func watchOps(op fsnotify.Op) *MapSet {
	ops := EmptySet()
	if op&fsnotify.Create != 0 {
		ops.Add(MakeKeyword("create"))
	}
	if op&fsnotify.Write != 0 {
		ops.Add(MakeKeyword("write"))
	}
	if op&fsnotify.Remove != 0 {
		ops.Add(MakeKeyword("remove"))
	}
	if op&fsnotify.Rename != 0 {
		ops.Add(MakeKeyword("rename"))
	}
	if op&fsnotify.Chmod != 0 {
		ops.Add(MakeKeyword("chmod"))
	}
	return ops
}
