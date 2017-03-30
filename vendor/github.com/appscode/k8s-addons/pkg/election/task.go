package election

import (
	"errors"
	"sync"
)

type Task interface {
	Run(string)
}

type taskHolder struct {
	sync.Mutex
	tasks map[string]Task
}

var defaultTasks = &taskHolder{
	tasks: make(map[string]Task),
}

func SetTask(name string, t Task) error {
	defaultTasks.Lock()
	defer defaultTasks.Unlock()

	if _, ok := defaultTasks.tasks[name]; ok {
		return errors.New("Task alreday exissts")
	}

	defaultTasks.tasks[name] = t
	return nil
}

func GetTask(name string) (Task, error) {
	defaultTasks.Lock()
	defer defaultTasks.Unlock()

	if t, ok := defaultTasks.tasks[name]; ok {
		return t, nil
	}
	return nil, errors.New("No task found")
}
