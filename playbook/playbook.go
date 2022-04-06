package playbook

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcmap/worker"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func NewPlaybook() *Playbook {
	return &Playbook{}
}

func (self MethodT) CanExecute() bool {
	return self.CanExecuteShell() || self.CanCallAPI()
}

func (self MethodT) CanExecuteShell() bool {
	return self.Shell != nil && self.Shell.Cmd != ""
}

func (self MethodT) CanCallAPI() bool {
	return self.API != nil && self.API.Urlstr != ""
}

func (self MethodT) ExecuteShell(req *worker.WorkerRequest, methodName string) (interface{}, error) {
	msg := req.Msg
	var ctx context.Context
	var cancel func()
	if self.Shell.Timeout != nil {
		ctx, cancel = context.WithTimeout(
			context.Background(),
			time.Second*time.Duration(*self.Shell.Timeout))
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", self.Shell.Cmd)

	cmd.Env = append(os.Environ(), self.Shell.Env...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()

	msgJson := jsonz.MessageString(msg)
	io.WriteString(stdin, msgJson)
	stdin.Close()

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if cmd.Process != nil {
		msg.Log().Infof("command for %s received output, pid %#v", methodName, cmd.Process.Pid)
	}
	var parsed interface{}
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (self MethodT) CallAPI(req *worker.WorkerRequest, methodName string) (interface{}, error) {
	api := self.API
	// api is not nil
	if api.client == nil {
		client, err := jsonzhttp.NewClient(api.Urlstr)
		if err != nil {
			return nil, err
		}
		if api.Header != nil && len(api.Header) > 0 {
			h := http.Header{}
			for k, v := range api.Header {
				h.Add(k, v)
			}
			client.SetExtraHeader(h)
		}
		api.client = client
	}

	var ctx context.Context
	var cancel func()

	if api.Timeout != nil {
		ctx, cancel = context.WithTimeout(
			context.Background(),
			time.Second*time.Duration(*api.Timeout))
		defer cancel()
	} else {
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	msg := req.Msg
	if msg.IsRequest() {
		reqmsg, _ := msg.(*jsonz.RequestMessage)
		resmsg, err := api.client.Call(ctx, reqmsg)
		return resmsg, err
	} else {
		msg.Log().Infof("send to url %s", api.Urlstr)
		err := api.client.Send(ctx, msg)
		return nil, err
	}
}

func (self *Playbook) Run(rootCtx context.Context, serverAddress string) error {
	w := worker.NewServiceWorker([]string{serverAddress})

	for name, method := range self.Config.Methods {
		if !method.CanExecute() {
			log.Warnf("cannot exec method %s %+v %s\n", name, method, method.Shell.Cmd)
			continue
		}
		log.Infof("playbook register %s", name)
		opts := make([]worker.WorkerHandlerSetter, 0)
		if method.innerSchema != nil {
			opts = append(opts, worker.WithSchema(method.innerSchema))
		}

		w.On(name, func(req *worker.WorkerRequest, params []interface{}) (interface{}, error) {
			req.Msg.Log().Infof("begin exec %s", name)
			var v interface{}
			var err error
			if method.CanExecuteShell() {
				v, err = method.ExecuteShell(req, name)
			} else {
				v, err = method.CallAPI(req, name)
			}
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					req.Msg.Log().Warnf(
						"command exit, code: %d, stderr: %s",
						exitErr.ExitCode(),
						string(exitErr.Stderr)[:100])
					return nil, jsonz.ErrLiveExit
				}

				req.Msg.Log().Warnf("error exec %s, %s", name, err.Error())
			} else {
				req.Msg.Log().Infof("end exec %s", name)
			}
			return v, err
		}, opts...)
	}

	w.ConnectWait(rootCtx)
	return nil
}
