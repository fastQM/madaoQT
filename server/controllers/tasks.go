package controllers

import (
	"encoding/json"
	"io/ioutil"
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
const DEBUG = true

func (t *TaskController) authen() (bool, iris.Map) {
	if DEBUG {
		return true, iris.Map{}
	}
	{
		s := t.Sessions.Start(t.Ctx)
		username := s.Get("name")
		if username == nil || username == "" {
			result := iris.Map{
				"result": false,
				"error":  errorCodeInvalidSession,
			}
			return false, result
		}
		return true, iris.Map{}
	}

}

// Get route: /task
func (t *TaskController) Get() iris.Map {

	// info := map[string]string{}
	// err := t.Ctx.ReadJSON(&info)
	// if err != nil {
	// 	Logger.Errorf("Error:%v", err)
	// 	t.Ctx.StatusCode(iris.StatusInternalServerError)
	// 	return nil
	// }

	// Logger.Infof("Data:%v", info)

	var tasksInfo []map[string]string
	t.Tasks.Range(func(key, value interface{}) bool {
		Logger.Infof("KEY:%s", key)
		defaultConfig, _ := json.Marshal(value.(Task.ITask).GetDefaultConfig())
		taskInfo := map[string]string{
			"name":    key.(string),
			"default": string(defaultConfig),
		}
		tasksInfo = append(tasksInfo, taskInfo)
		return true
	})

	configBytes, _ := json.Marshal(tasksInfo)

	return iris.Map{
		"result": true,
		"tasks":  string(configBytes),
	}
}

func (t *TaskController) PostStart() iris.Map {

	var errMsg string
	if ok, result := t.authen(); !ok {
		return result
	}

	if task, ok := t.Tasks.Load("okexdiff"); ok {
		body, err := ioutil.ReadAll(t.Ctx.Request().Body)
		if err != nil {
			Logger.Debugf("fail to read:%v", err)
			return iris.Map{
				"result": false,
				"error":  errorCodeInvalidParameters,
			}
		}

		err = task.(Task.ITask).Start(string(body))
		if err != nil {
			errMsg = err.Error()
			goto _ERROR
		}

		return iris.Map{
			"result": true,
		}
	}

_ERROR:
	return iris.Map{
		"result": false,
		"error":  errMsg,
	}
}

func (t *TaskController) GetStop() iris.Map {

	if ok, result := t.authen(); !ok {
		return result
	}

	if task, ok := t.Tasks.Load("okexdiff"); ok {
		task.(Task.ITask).Close()
		return iris.Map{
			"result": true,
		}
	}

	return iris.Map{
		"result": false,
	}
}

// func (t *TaskController) GetRun() iris.Map {

// 	if ok, result := t.authen(); !ok {
// 		return result
// 	}

// 	value, ok := t.Tasks.Load("okexdiff")
// 	if ok {
// 		task := value.(*Task.Task)
// 		task.InstallTaskAndRun(task.Name, "hello")
// 		return iris.Map{
// 			"result": true,
// 		}
// 	}

// 	return iris.Map{
// 		"result": false,
// 	}

// }

// func (t *TaskController) GetStop() iris.Map {

// 	if ok, result := t.authen(); !ok {
// 		return result
// 	}

// 	value, ok := t.Tasks.Load("okexdiff")
// 	if ok {
// 		task := value.(*Task.Task)
// 		task.ExitTask()
// 		return iris.Map{
// 			"result": true,
// 		}
// 	}

// 	return iris.Map{
// 		"result": false,
// 	}

// }
