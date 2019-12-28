package engine

import (
	"flag"

	"github.com/takama/daemon"
)

func runDaemon(runner func() error) error {
	service, err := daemon.New("vince", "nginx compatible http server")
	if err != nil {
		return err
	}
	a := flag.Args()
	usage := "Usage: vince install | remove | start | stop | status"
	if len(a) > 0 {
		switch a[0] {
		case "install":
			u, err := service.Install()
			if err != nil {
				return err
			}
			println(u)
			return nil
		case "remove":
			u, err := service.Remove()
			if err != nil {
				return err
			}
			println(u)
			return nil
		case "start":
			u, err := service.Start()
			if err != nil {
				return err
			}
			println(u)
			return nil
		case "stop":
			u, err := service.Stop()
			if err != nil {
				return err
			}
			println(u)
			return nil
		case "status":
			u, err := service.Status()
			if err != nil {
				return err
			}
			println(u)
			return nil
		default:
			println(usage)
			return nil
		}
	}
	return runner()
}
