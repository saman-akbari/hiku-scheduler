package worker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/server"

	"github.com/urfave/cli/v2"
)

// initCmd corresponds to the "init" command of the admin tool.
func initCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	if err := common.LoadDefaults(olPath); err != nil {
		return err
	}

	if err := initOLDir(olPath, ctx.String("image"), ctx.Bool("newbase")); err != nil {
		return err
	}
	fmt.Printf("\nYou may optionally modify the defaults here: %s\n\n",
		filepath.Join(olPath, "config.json"))
	fmt.Printf("Next start a worker using the \"ol worker up\" command.\n")
	return nil
}

// upCmd corresponds to the "up" command of the admin tool.
func upCmd(ctx *cli.Context) error {
	// get path of worker files
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	// PREP STEP 1: make sure we have a worker directory
	if _, err := os.Stat(olPath); os.IsNotExist(err) {
		// need to init worker dir first
		fmt.Printf("Did not find OL directory at %s\n", olPath)
		if err := common.LoadDefaults(olPath); err != nil {
			return err
		}

		if err := initOLDir(olPath, ctx.String("image"), false); err != nil {
			return err
		}
	}

	// PREP STEP 2: load config file and apply any command-line overrides
	confPath := filepath.Join(olPath, "config.json")
	overrides := ctx.String("options")
	if overrides != "" {
		overridesPath := confPath + ".overrides"
		err = overrideOpts(confPath, overridesPath, overrides)
		if err != nil {
			return err
		}
		confPath = overridesPath
	}
	if err := common.LoadConf(confPath); err != nil {
		return err
	}

	// PREP STEP 3: stop any prior worker that may be running
	if err := stopOL(olPath); err != nil {
		return err
	}

	// should we run as a background process?
	detach := ctx.Bool("detach")

	if detach {
		// stdout+stderr both go to log
		logPath := filepath.Join(olPath, "worker.out")
		// creates a worker.out file
		f, err := os.Create(logPath)
		if err != nil {
			return err
		}
		// holds attributes that will be used when os.StartProcess.
		// we use CLONE_NEWNS because ol creates many mount points.
		// we don't want them to show up in /proc/self/mountinfo
		// for systemd because systemd creates a service for each
		// mount point, which is a major overhead.
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
			Sys: &syscall.SysProcAttr{
				Unshareflags: syscall.CLONE_NEWNS,
			},
		}
		cmd := []string{}
		for _, arg := range os.Args {
			if arg != "-d" && arg != "--detach" {
				cmd = append(cmd, arg)
			}
		}
		// looks for ./ol path
		binPath, err := exec.LookPath(os.Args[0])
		if err != nil {
			return err
		}
		// start the worker process
		fmt.Printf("Starting worker in %s and waiting until it's ready.\n", olPath)
		proc, err := os.StartProcess(binPath, cmd, &attr)
		if err != nil {
			return err
		}

		// died is error message
		died := make(chan error)
		go func() {
			_, err := proc.Wait()
			died <- err
		}()

		fmt.Printf("\tPID: %d\n\tPort: %s\n\tLog File: %s\n", proc.Pid, common.Conf.Worker_port, logPath)

		var pingErr error

		for i := 0; i < 300; i++ {
			// check if it has died
			select {
			case err := <-died:
				if err != nil {
					return err
				}
				return fmt.Errorf("worker process %d does not a appear to be running, check worker.out", proc.Pid)
			default:
			}

			// is the worker still alive?
			err := proc.Signal(syscall.Signal(0))
			if err != nil {

			}

			// is it reachable?
			url := fmt.Sprintf("http://%s:%s/pid", common.Conf.Worker_url, common.Conf.Worker_port)
			response, err := http.Get(url)
			if err != nil {
				pingErr = err
				time.Sleep(100 * time.Millisecond)
				continue
			}
			defer response.Body.Close()

			// are we talking with the expected PID?
			body, err := ioutil.ReadAll(response.Body)
			pid, err := strconv.Atoi(strings.TrimSpace(string(body)))
			if err != nil {
				return fmt.Errorf("/pid did not return an int :: %s", err)
			}

			if pid == proc.Pid {
				fmt.Printf("Ready!\n")
				return nil // server is started and ready for requests
			}

			return fmt.Errorf("expected PID %v but found %v (port conflict?)", proc.Pid, pid)
		}

		return fmt.Errorf("worker still not reachable after 30 seconds :: %s", pingErr)
	}

	if err := server.Main(); err != nil {
		return err
	}

	return fmt.Errorf("this code should not be reachable")
}

// status corresponds to the "status" command of the admin tool.
func statusCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	err = common.LoadConf(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}

	fmt.Printf("Worker Ping:\n")
	url := fmt.Sprintf("http://%s:%s/status", common.Conf.Worker_url, common.Conf.Worker_port)
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not send GET to %s", url)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read body from GET to %s", url)
	}
	fmt.Printf("  %s => %s [%s]\n", url, body, response.Status)
	fmt.Printf("\n")

	return nil
}

// down corresponds to the "down" command of the admin tool.
func downCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}
	err = common.LoadConf(filepath.Join(olPath, "config.json"))
	if err != nil {
		return err
	}
	return stopOL(olPath)
}

// cleanup corresponds to the "force-cleanup" command of the admin tool.
func cleanupCmd(ctx *cli.Context) error {
	olPath, err := common.GetOlPath(ctx)
	if err != nil {
		return err
	}

	cgRoot := filepath.Join("/sys", "fs", "cgroup", filepath.Base(olPath)+"-sandboxes")
	fmt.Printf("ATTEMPT to cleanup cgroups at %s\n", cgRoot)

	if files, err := ioutil.ReadDir(cgRoot); err != nil {
		fmt.Printf("could not find cgroup root: %s\n", err.Error())
	} else {
		kill := filepath.Join(cgRoot, "cgroup.kill")
		if err := ioutil.WriteFile(kill, []byte(fmt.Sprintf("%d", 1)), os.ModeAppend); err != nil {
			fmt.Printf("could kill processes in cgroup: %s\n", err.Error())
		}

		for _, file := range files {
			if strings.HasPrefix(file.Name(), "cg-") {
				cg := filepath.Join(cgRoot, file.Name())
				fmt.Printf("try removing %s\n", cg)
				if err := syscall.Rmdir(cg); err != nil {
					fmt.Printf("could remove cgroup: %s\n", err.Error())
				}
			}
		}

		if err := syscall.Rmdir(cgRoot); err != nil {
			fmt.Printf("could remove cgroup root: %s\n", err.Error())
		}
	}

	dirName := filepath.Join(olPath, "worker", "root-sandboxes")
	fmt.Printf("ATTEMPT to cleanup mounts at %s\n", dirName)

	if files, err := ioutil.ReadDir(dirName); err != nil {
		fmt.Printf("could not find mount root: %s\n", err.Error())
	} else {
		for _, file := range files {
			path := filepath.Join(dirName, file.Name())
			fmt.Printf("try unmounting %s\n", path)
			if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
				fmt.Printf("could not unmount: %s\n", err.Error())
			}

			if err := syscall.Rmdir(path); err != nil {
				fmt.Printf("could remove mount dir: %s\n", err.Error())
			}
		}
	}

	if err := syscall.Unmount(dirName, syscall.MNT_DETACH); err != nil {
		fmt.Printf("could not unmount %s: %s\n", dirName, err.Error())
	}

	if err := os.Remove(filepath.Join(olPath, "worker", "worker.pid")); err != nil {
		fmt.Printf("could not remove worker.pid: %s\n", err.Error())
	}

	return nil
}

func WorkerCommands() []*cli.Command {
	pathFlag := cli.StringFlag{
		Name:    "path",
		Aliases: []string{"p"},
		Usage:   "Path location for OL environment",
	}
	dockerImgFlag := cli.StringFlag{
		Name:    "image",
		Aliases: []string{"i"},
		Usage:   "Name of Docker image to use for base",
	}

	cmds := []*cli.Command{
		&cli.Command{
			Name:        "init",
			Usage:       "Create an OL worker environment, including default config and dump of base image",
			UsageText:   "ol init [OPTIONS...]",
			Description: "A cluster directory of the given name will be created with internal structure initialized.",
			Flags: []cli.Flag{
				&pathFlag,
				&dockerImgFlag,
				&cli.BoolFlag{
					Name:    "newbase",
					Aliases: []string{"b"},
					Usage:   "Overwrite base directory if it already exists",
				},
			},
			Action: initCmd,
		},
		&cli.Command{
			Name:        "up",
			Usage:       "Start an OL worker process (automatically calls 'init' and uses default if that wasn't already done)",
			UsageText:   "ol up [OPTIONS...] [--detach]",
			Description: "Start an OL worker.",
			Flags: []cli.Flag{
				&pathFlag,
				&dockerImgFlag,
				&cli.StringFlag{
					Name:    "options",
					Aliases: []string{"o"},
					Usage:   "Override options with: -o opt1=val1,opt2=val2/opt3.subopt31=val3",
				},
				&cli.BoolFlag{
					Name:    "detach",
					Aliases: []string{"d"},
					Usage:   "Run worker in background",
				},
			},
			Action: upCmd,
		},
		&cli.Command{
			Name:      "down",
			Usage:     "Kill containers and processes of the worker",
			UsageText: "ol down [OPTIONS...]",
			Flags:     []cli.Flag{&pathFlag},
			Action:    downCmd,
		},
		&cli.Command{
			Name:        "status",
			Usage:       "check status of an OL worker process",
			UsageText:   "ol status [OPTIONS...]",
			Description: "If no cluster name is specified, number of containers of each cluster is printed; otherwise the connection information for all containers in the given cluster will be displayed.",
			Flags:       []cli.Flag{&pathFlag},
			Action:      statusCmd,
		},
		&cli.Command{
			Name:      "force-cleanup",
			Usage:     "Developer use only.  Cleanup cgroups and mount points (only needed when OL halted unexpectedly or there's a bug)",
			UsageText: "ol force-cleanup [OPTIONS...]",
			Flags:     []cli.Flag{&pathFlag},
			Action:    cleanupCmd,
		},
	}

	return cmds
}
