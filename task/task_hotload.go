package task

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

/*
	实时加载的任务；目前暂不考虑支持
*/

type TaskHotLoad struct {
	// GetTaskExplanation() *TaskExplanation
	Name  string
	Paras string
	cmd   *exec.Cmd
}

func (t *TaskHotLoad) InstallTaskAndRun(name string, paramters string) error {

	var path = "madaoQT/task/"
	cmd := exec.Command("go", "install", path+name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		Logger.Errorf("Fail to install:%v", err)
		return errors.New(string(out))
	}

	cmd = exec.Command(name, "-config="+paramters)
	if cmd == nil {
		return errors.New("Fail to run task")
	}
	// Logger.Infof("Task Command:%v", cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// cmd.Stdin = os.Stdin
	cmd.Start()
	Logger.Infof("Task Command:%v, Task ID:%v", cmd.Args, cmd.Process.Pid)

	t.cmd = cmd
	return nil
}

func (t *TaskHotLoad) ExitTask() {
	if t.cmd == nil {
		Logger.Errorf("Invalid command to Exit")
		return
	}
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()
	Logger.Infof("Exiting task:%v", t.cmd.Process.Pid)
	select {
	case <-time.After(1 * time.Second):
		/*
			We would like to kill the process by signal, but there maybe some problem in windows; So we will use websocket to send the signal
		*/
		if err := t.cmd.Process.Kill(); err != nil {
			Logger.Errorf("Fail to kill task:%v", err)
		}
		Logger.Info("Succeed to kill task")
	case err := <-done:
		if err != nil {
			Logger.Errorf("Task exit with the error %v", err)
		}
	}
}
