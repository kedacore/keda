package configs

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/olekukonko/tablewriter"
	"go.uber.org/zap"
)

func GetRootRepositoryPath() string {
	out, err := exec.Command("sh", "-c", "git rev-parse --show-toplevel").Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSuffix(string(out), "\n")
}

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), err
}

// HumanDuration returns a human-readable approximation of a duration
// (eg. "About a minute", "4 hours ago", etc.).
func HumanDuration(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func IsUrl(str string) bool {
	urlStr, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	address := net.ParseIP(urlStr.Host)

	if address == nil {
		return strings.Contains(urlStr.Host, ".")
	}

	return true
}

func ConvertDurationToStr(d time.Duration) (result string) {
	if int64(d/time.Second) <= 60 {
		result = fmt.Sprintf("%ds", int64(d/time.Second))
	} else if int64(d/time.Minute) <= 60 {
		result = fmt.Sprintf("%dm", int64(d/time.Minute))
	} else if int64(d/time.Hour) <= 60 {
		result = fmt.Sprintf("%dh", int64(d/time.Hour))
	} else if int64(d/(time.Hour)) >= 24 {
		result = fmt.Sprintf("%dd", int64(d/(time.Hour*24)))
	}

	return result
}

var onlyOneSignalHandler = make(chan struct{})

func SetupSignalHandler(l ...interface{}) context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGABRT)

	go func() {
		<-c

		closeAll(l...)
		cancel()

		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}

func closeAll(l ...interface{}) {
	for _, cl := range l {
		switch closer := cl.(type) {
		case SignalStopper:
			closer.Stop()
		case SignalCloser:
			closer.Close()
		case SignalCloserWithErr:
			err := closer.Close()
			if err != nil {
				switch logger := l[len(l)-1].(type) {
				case *zap.SugaredLogger:
					logger.Errorf("🔥 close SignalCloserWithErr object type: %T, error: %v", closer, err)
				case *log.Logger:
					logger.Printf("🔥 close SignalCloserWithErr object type: %T, error: %v", closer, err)
				}
			}
		case SignalStopperWithErr:
			err := closer.Stop()
			if err != nil {
				switch logger := l[len(l)-1].(type) {
				case *zap.SugaredLogger:
					logger.Errorf("🔥 stop SignalStopperWithErr object type: %T, error: %v", closer, err)
				case *log.Logger:
					logger.Printf("🔥 stop SignalStopperWithErr object type: %T, error: %v", closer, err)
				}
			}
		}
	}
}

var doOnce sync.Once

func OsSignalHandler(cancel context.CancelFunc, callbacks []func(), l ...interface{}) {
	doOnce.Do(func() {
		var Stop = make(chan os.Signal, 1)

		signal.Notify(Stop,
			syscall.SIGTERM,
			syscall.SIGINT,
			syscall.SIGABRT,
		)

		for range Stop {
			if cancel != nil {
				cancel()
			}

			closeAll(l...)

			for _, callback := range callbacks {
				callback()
			}

			return
		}
	})

	//ctx, cancel := context.WithCancel(context.Background())
}

const (
	MonitoringKey = "monitoring"
	ProfilingKey  = "profiling"
)

func JoinClosers(in map[string]SignalCloserWithErr, conf *Base) (out []SignalCloserWithErr) {
	for k, v := range in {
		if k == MonitoringKey {
			if conf.Monitoring.Enabled && v != nil {
				out = append(out, v)
			}

			continue
		}

		if k == ProfilingKey {
			if conf.Profiling.Enabled && v != nil {
				out = append(out, v)
			}

			continue
		}

		out = append(out, v)
	}

	return out
}

func PrintFlags() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage: service (%s) [options] param>:\n", os.Args[0])

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Flag name", "Flag Value", "Flag description"})
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	counter := 0
	flag.VisitAll(func(f *flag.Flag) {
		table.Append([]string{f.Name, fmt.Sprintf("%v", f.Value), fmt.Sprintf("%v", f.Usage)})
		if counter%3 == 0 {
			table.SetRowLine(true)
		}
		counter++
	})
	table.Render()
}
