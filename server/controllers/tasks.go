package controllers

import (
	"encoding/json"
	"sync"

	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"github.com/kataras/iris/sessions"

	Task "madaoQT/task"
)

type TaskController struct {
	mvc.C

	// [ Your fields here ]
	// Request lifecycle data
	// Models
	// Database
	// Global properties
	Sessions *sessions.Sessions `iris:"persistence"`
	Tasks    *sync.Map          `iris:"persistence"`
}

// GET /task/tasks

func (t *TaskController) authen() (bool, iris.Map) {
	s := t.Sessions.Start(t.Ctx)
	username := s.Get("name")
	if username == nil || username == "" {
		result := iris.Map{
			"result": false,
			"error":  errorCodeInvalidSession,
		}
		return false, result
	}
	return false, iris.Map{}
}

// Get route: /task
func (t *TaskController) Post() iris.Map {

	info := map[string]string{}
	err := t.Ctx.ReadJSON(&info)
	if err != nil {
		Logger.Errorf("Error:%v", err)
		t.Ctx.StatusCode(iris.StatusInternalServerError)
		return nil
	}

	Logger.Infof("Data:%v", info)

	var tasksInfo []string
	t.Tasks.Range(func(key, value interface{}) bool {
		Logger.Infof("KEY:%s", key)
		tasksInfo = append(tasksInfo, key.(string))
		return true
	})

	configBytes, _ := json.Marshal(tasksInfo)

	return iris.Map{
		"result": true,
		"tasks":  string(configBytes),
	}
}

func (t *TaskController) GetRun() iris.Map {

	if ok, result := t.authen(); !ok {
		return result
	}

	value, ok := t.Tasks.Load("okexdiff")
	if ok {
		task := value.(*Task.Task)
		task.InstallTaskAndRun(task.Name, "hello")
		return iris.Map{
			"result": true,
		}
	}

	return iris.Map{
		"result": false,
	}

}

func (t *TaskController) GetStop() iris.Map {

	if ok, result := t.authen(); !ok {
		return result
	}

	value, ok := t.Tasks.Load("okexdiff")
	if ok {
		task := value.(*Task.Task)
		task.ExitTask()
		return iris.Map{
			"result": true,
		}
	}

	return iris.Map{
		"result": false,
	}

}
