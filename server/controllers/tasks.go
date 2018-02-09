package controllers

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"

	Mongo "madaoQT/mongo"
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

	// configBytes, _ := json.Marshal(tasksInfo)

	return iris.Map{
		"result": true,
		"tasks":  tasksInfo,
	}
}

func (t *TaskController) PostStart() iris.Map {

	var errMsg string
	var mongo *Mongo.ExchangeDB

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

		mongo = new(Mongo.ExchangeDB)
		if err = mongo.Connect(); err != nil {
			return iris.Map{
				"result": false,
				"error":  errorCodeMongoDisconnect,
			}
		}

		err, record := mongo.FindOne("OkexSpot")
		if err != nil {
			return iris.Map{
				"result": false,
				"error":  err.Error(),
			}
		} else if record == nil {
			return iris.Map{
				"result": false,
				"error":  errorCodeAPINotSet,
			}
		}

		err = task.(Task.ITask).Start(record.API, record.Secret, string(body))
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

func (t *TaskController) GetStatus() iris.Map {

	if task, ok := t.Tasks.Load("okexdiff"); ok {
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

func (t *TaskController) GetBalances() iris.Map {

	if task, ok := t.Tasks.Load("okexdiff"); ok {
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

func (t *TaskController) GetPositions() iris.Map {

	if task, ok := t.Tasks.Load("okexdiff"); ok {
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
