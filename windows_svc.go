package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tools_local_mix_proxy/internal/cmd"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/marcellowy/go-common/gogf/vlog"
	"github.com/marcellowy/go-common/tools"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type WinService struct {
	ServiceName               string
	ServiceDisplayName        string
	ServiceDescription        string
	AutoStart                 bool
	ForceReinstallOnDuplicate bool
}

func (*WinService) StartApp() {
	cmd.Start()
}

func (*WinService) StopApp() {
	cmd.Stop()
}

func (w *WinService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	go w.StartApp() // start app
	changes <- svc.Status{State: svc.Running, Accepts: cmdAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				w.StopApp() // stop app
				changes <- svc.Status{State: svc.Stopped}
				break loop
			}
		}
	}
	return false, 0
}

// Install 安装
func (w *WinService) Install(ctx context.Context) (err error) {
	var exePath string
	exePath, err = os.Executable()
	if err != nil {
		vlog.Errorf(ctx, "Failed to get executable path: %v", err)
		return
	}
	if exePath, err = filepath.Abs(exePath); err != nil {
		vlog.Errorf(ctx, "Failed to get absolute path of executable: %v", err)
		return
	}
	var m *mgr.Mgr
	if m, err = mgr.Connect(); err != nil {
		vlog.Errorf(ctx, "Failed to connect to service manager. %v", err)
		return
	}
	defer func() {
		_ = m.Disconnect()
	}()
	var services []string
	if services, err = m.ListServices(); err != nil {
		vlog.Error(ctx, err)
		return
	}
	for _, service := range services {
		if service == w.ServiceName {
			if w.ForceReinstallOnDuplicate {
				if err = w.Uninstall(ctx); err != nil {
					vlog.Error(ctx, err)
					return
				}
				time.Sleep(3 * time.Second)
			} else {
				err = fmt.Errorf("service %s is already installed", service)
				vlog.Warning(ctx, err)
				return
			}
			break
		}
	}
	var startType uint32 = mgr.StartManual
	if w.AutoStart {
		startType = mgr.StartAutomatic
	}
	var s *mgr.Service
	s, err = m.CreateService(w.ServiceName, exePath, mgr.Config{
		DisplayName: w.ServiceDisplayName,
		Description: w.ServiceDescription,
		StartType:   startType,
	})
	if err != nil {
		vlog.Errorf(ctx, "Failed to create service %s: %v", w.ServiceName, err)
		return
	}
	defer tools.Close(s)
	return
}

func (w *WinService) Uninstall(ctx context.Context) (err error) {
	var m *mgr.Mgr
	if m, err = mgr.Connect(); err != nil {
		vlog.Error(ctx, err)
		return
	}
	defer func() {
		_ = m.Disconnect()
	}()
	var s *mgr.Service
	if s, err = m.OpenService(w.ServiceName); err != nil {
		vlog.Error(ctx, err)
		return
	}
	defer tools.Close(s)
	var status svc.Status
	if status, err = s.Query(); err != nil {
		vlog.Error(ctx, err)
		return
	}
	if status.State == svc.Running {
		if _, err = s.Control(svc.Stop); err != nil {
			vlog.Error(ctx, err)
			return
		}
		time.Sleep(time.Second * 1)
	}
	if err = s.Delete(); err != nil {
		vlog.Error(ctx, err)
	}
	return
}

func main() {
	var (
		ctx    = gctx.New()
		err    error
		server = &WinService{
			ServiceName:               "AATestGoServer",
			ServiceDisplayName:        "AATestGoServer 服务",
			ServiceDescription:        "AATestGoServer 描述",
			AutoStart:                 true,
			ForceReinstallOnDuplicate: true,
		}
	)
	vlog.Info(ctx, "================== start ==================")
	if len(os.Args) > 1 {
		action := strings.ToLower(os.Args[1])
		vlog.Info(ctx, "cli action:", action)
		switch action {
		case "install":
			if err = server.Install(ctx); err != nil {
				vlog.Error(ctx, err)
				return
			}
			vlog.Info(ctx, "install success")
			return
		case "uninstall":
			if err = server.Uninstall(ctx); err != nil {
				vlog.Error(ctx, err)
				return
			}
			vlog.Info(ctx, "uninstall success")
			return
		default:
			vlog.Warningf(ctx, "unknown action: %s", action)
		}
		return
	}
	if ok, _ := svc.IsWindowsService(); ok {
		if err = svc.Run(server.ServiceName, &WinService{}); err != nil {
			vlog.Error(ctx, err)
			return
		}
		return
	}
	server.StartApp()
}
