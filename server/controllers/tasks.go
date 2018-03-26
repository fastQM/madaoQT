package controllers

import (
	"io/ioutil"
	"sync"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"

	Task "madaoQT/task"
)

type TaskController struct {
	Ctx iris.Context

	// [ Your fields here ]
	// Request lifecycle data
	// Models
	// Database
	// Global properties
	Sessions *sessions.Sessions `iris:"persistence"`
	Tasks    *sync.Map          `iris:"persistence"`
}

// 管理员接口

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

// GetList 获取任务列表
// Get route: /task/list
func (t *TaskController) GetList() iris.Map {
	var tasks []Task.Description
	t.Tasks.Range(func(key, value interface{}) bool {
		Logger.Infof("KEY:%s", key)
		task := value.(Task.ITask)
		tasks = append(tasks, task.GetDescription())
		return true
	})

	return iris.Map{
		"result": true,
		"data":   tasks,
	}

}

// Get route: /task
func (t *TaskController) GetBy(name string) iris.Map {

	// info := map[string]string{}
	// err := t.Ctx.ReadJSON(&info)
	// if err != nil {
	// 	Logger.Errorf("Error:%v", err)
	// 	t.Ctx.StatusCode(iris.StatusInternalServerError)
	// 	return nil
	// }

	// Logger.Infof("Data:%v", info)

	t.Ctx.ViewData("task", name)
	t.Ctx.View("task.html")

	return nil
	// var tasksInfo []map[string]string
	// t.Tasks.Range(func(key, value interface{}) bool {
	// 	Logger.Infof("KEY:%s", key)
	// 	defaultConfig, _ := json.Marshal(value.(Task.ITask).GetDefaultConfig())
	// 	taskInfo := map[string]string{
	// 		"name":    key.(string),
	// 		"default": string(defaultConfig),
	// 	}
	// 	tasksInfo = append(tasksInfo, taskInfo)
	// 	return true
	// })

	// // configBytes, _ := json.Marshal(tasksInfo)

	// return iris.Map{
	// 	"result": true,
	// 	"tasks":  tasksInfo,
	// }
}

// PostStart 启动任务
// Post route: /task/start
func (t *TaskController) PostStartBy(name string) iris.Map {

	var errMsg string

	if ok, result := t.authen(); !ok {
		return result
	}

	Logger.Infof("Task %s", name)
	if name == "" {
		return iris.Map{
			"result": false,
			"error":  errorCodeInvalidParameters,
		}
	}

	if task, ok := t.Tasks.Load(name); ok {
		body, err := ioutil.ReadAll(t.Ctx.Request().Body)
		if err != nil {
			Logger.Debugf("fail to read:%v", err)
			return iris.Map{
				"result": false,
				"error":  errorCodeInvalidParameters,
			}
		}

		err = task.(Task.ITask).Start(string(body))
		Logger.Errorf("Error:%v", err)
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

func (t *TaskController) GetStatusBy(name string) iris.Map {

	Logger.Infof("Task %s", name)
	if name == "" {
		return iris.Map{
			"result": false,
			"error":  errorCodeInvalidParameters,
		}
	}

	if task, ok := t.Tasks.Load(name); ok {
		result := task.(Task.ITask).GetStatus()
		return iris.Map{
			"result": true,
			"data":   result,
		}
	}

	return iris.Map{
		"result": false,
	}
}

func (t *TaskController) GetBalancesBy(name string) iris.Map {

	Logger.Infof("Task %s", name)
	if name == "" {
		return iris.Map{
			"result": false,
			"error":  errorCodeInvalidParameters,
		}
	}

	if task, ok := t.Tasks.Load(name); ok {
		result := task.(Task.ITask).GetBalances()
		return iris.Map{
			"result": true,
			"data":   result,
		}
	}

	return iris.Map{
		"result": false,
	}
}

func (t *TaskController) GetTrades() iris.Map {

	if task, ok := t.Tasks.Load("okexdiff"); ok {
		result := task.(Task.ITask).GetTrades()
		// Logger.Debugf("getTrades:%v", result)
		return iris.Map{
			"result": true,
			"data":   result,
		}
	}

	return iris.Map{
		"result": false,
	}
}

func (t *TaskController) GetPositionsBy(name string) iris.Map {

	Logger.Infof("GetPositionsBy %s", name)
	if name == "" {
		return iris.Map{
			"result": false,
			"error":  errorMessage[errorCodeInvalidParameters],
		}
	}

	if task, ok := t.Tasks.Load(name); ok {

		if task.(Task.ITask).GetStatus() != Task.StatusProcessing {
			return iris.Map{
				"result": false,
				"error":  errorMessage[errorCodeTaskNotRunning],
			}
		}

		result := task.(Task.ITask).GetPositions()
		if result != nil {
			Logger.Debugf("GetPositions:%v", result)
			return iris.Map{
				"result": true,
				"data":   result,
			}
		}
	}

	return iris.Map{
		"result": false,
	}
}

func (t *TaskController) GetFailed() iris.Map {
	if task, ok := t.Tasks.Load("okexdiff"); ok {
		result := task.(Task.ITask).GetFailedPositions()
		if result != nil {
			Logger.Debugf("GetFailedPositions:%v", result)
			return iris.Map{
				"result": true,
				"data":   result,
			}
		}
	}

	return iris.Map{
		"result": false,
	}
}

func (t *TaskController) PostFix() iris.Map {

	if task, ok := t.Tasks.Load("okexdiff"); ok {
		body, err := ioutil.ReadAll(t.Ctx.Request().Body)
		if err != nil {
			Logger.Debugf("fail to read:%v", err)
			return iris.Map{
				"result": false,
				"error":  errorCodeInvalidParameters,
			}
		}

		Logger.Infof("Body:%v", string(body))

		result := task.(Task.ITask).FixFailedPosition(string(body))
		if result == nil {
			return iris.Map{
				"result": true,
			}
		} else {
			return iris.Map{
				"result": false,
				"error":  result.Error(),
			}
		}

	}

	return iris.Map{
		"result": false,
	}
}

func (t *TaskController) GetStop() iris.Map {

	// if ok, result := t.authen(); !ok {
	// 	return result
	// }

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
