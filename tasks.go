package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	. "github.com/candid82/joker/core"
)

func findTaskNamespace() *Namespace {
	if tasksNs := GLOBAL_ENV.FindNamespace(MakeSymbol("tasks")); tasksNs != nil {
		return tasksNs
	}
	return GLOBAL_ENV.CurrentNamespace()
}

func isRunnableTask(vr *Var) bool {
	if vr == nil || vr.IsPrivate() || vr.IsMacro() {
		return false
	}
	vl := vr.Resolve()
	if vl == nil {
		return false
	}
	_, ok := vl.(Callable)
	return ok
}

func getTaskDoc(vr *Var) string {
	if m := vr.GetMeta(); m != nil {
		if ok, v := m.Get(MakeKeyword("doc")); ok {
			if s, ok := v.(String); ok {
				lines := strings.Split(strings.TrimSpace(s.S), "\n")
				if len(lines) > 0 {
					return strings.TrimSpace(lines[0])
				}
			}
		}
	}
	return ""
}

func tryCall(c Callable, args []Object) (res Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case *EvalError:
				err = e
			case *ExInfo:
				err = e
			case error:
				err = e
			default:
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	return c.Call(args), nil
}

func resolveTask(taskName string) *Var {
	if strings.Contains(taskName, "/") {
		parts := strings.SplitN(taskName, "/", 2)
		if targetNs := GLOBAL_ENV.FindNamespace(MakeSymbol(parts[0])); targetNs != nil {
			return targetNs.Resolve(parts[1])
		}
		return nil
	}

	ns := findTaskNamespace()
	if ns != nil {
		if vr := ns.Resolve(taskName); vr != nil {
			return vr
		}
		if vr := ns.Resolve(strings.ReplaceAll(taskName, "_", "-")); vr != nil {
			return vr
		}
	}

	userNs := GLOBAL_ENV.FindNamespace(MakeSymbol("user"))
	if userNs != nil && userNs != ns {
		if vr := userNs.Resolve(taskName); vr != nil {
			return vr
		}
		if vr := userNs.Resolve(strings.ReplaceAll(taskName, "_", "-")); vr != nil {
			return vr
		}
	}

	return nil
}

func listTasks(tasksFile string) {
	if _, err := os.Stat(tasksFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(Stderr, "Error: Tasks file not found: %s\n", tasksFile)
		} else {
			fmt.Fprintf(Stderr, "Error: %s: %v\n", tasksFile, err)
		}
		ExitJoker(1)
	}

	if err := processFile(tasksFile, EVAL); err != nil {
		ExitJoker(1)
	}

	ns := findTaskNamespace()
	if ns == nil {
		fmt.Fprintf(Stderr, "Error: No tasks namespace found in %s\n", tasksFile)
		ExitJoker(1)
	}

	type taskInfo struct {
		name string
		doc  string
	}

	tasksMap := make(map[string]taskInfo)
	maxLen := 0

	addTasksFromNs := func(targetNs *Namespace) {
		for _, vr := range targetNs.Mappings() {
			if vr.GetNamespace() == targetNs && isRunnableTask(vr) {
				name := vr.Symbol().Name()
				doc := getTaskDoc(vr)
				if len(name) > maxLen {
					maxLen = len(name)
				}
				tasksMap[name] = taskInfo{name: name, doc: doc}
			}
		}
	}

	addTasksFromNs(ns)
	if len(tasksMap) == 0 && ns.Name.Name() == "tasks" {
		if userNs := GLOBAL_ENV.FindNamespace(MakeSymbol("user")); userNs != nil {
			addTasksFromNs(userNs)
		}
	}

	taskNames := make([]string, 0, len(tasksMap))
	for name := range tasksMap {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	for _, name := range taskNames {
		t := tasksMap[name]
		if t.doc != "" {
			fmt.Fprintf(Stdout, "%-*s  %s\n", maxLen, t.name, t.doc)
		} else {
			fmt.Fprintf(Stdout, "%s\n", t.name)
		}
	}
}

func runTask(tasksFile string, taskName string, taskArgs []string) error {
	if _, err := os.Stat(tasksFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(Stderr, "Error: Tasks file not found: %s\n", tasksFile)
		} else {
			fmt.Fprintf(Stderr, "Error: %s: %v\n", tasksFile, err)
		}
		return err
	}

	if err := processFile(tasksFile, EVAL); err != nil {
		return err
	}

	vr := resolveTask(taskName)
	if vr == nil || !isRunnableTask(vr) {
		err := fmt.Errorf("Task '%s' not found in %s", taskName, tasksFile)
		fmt.Fprintf(Stderr, "Error: %s\n", err)
		return err
	}

	args := make([]Object, len(taskArgs))
	for i, arg := range taskArgs {
		args[i] = MakeString(arg)
	}

	res, err := tryCall(vr, args)
	if err != nil {
		fmt.Fprintln(Stderr, err)
		return err
	}

	if intVal, ok := res.(Int); ok && intVal.I != 0 {
		ExitJoker(int(intVal.I))
	}
	return nil
}
