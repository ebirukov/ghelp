package linux

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

const (
	defaultSubsystem = "syscalls"
)

var BootstrapEvents TraceEvents = parseBootstrapEvents()

type TraceEvents []string

func (events TraceEvents) Enable() error {
	return events.setEnable(true)
}

func (events TraceEvents) Disable() error {
	return events.setEnable(false)
}

func (events TraceEvents) setEnable(enable bool) error {
	flag := "0"
	if enable {
		flag = "1"
	}

	for _, e := range events {
		se := strings.Split(e, ":")
		event := se[len(se)-1]
		subsystem := defaultSubsystem
		if len(se) > 1 {
			subsystem = se[0]
		}
		path := fmt.Sprintf("events/%s/%s/enable", subsystem, event)
		b, err := TraceFS.ReadFile(path)
		if err != nil {
			if !errors.Is(err, ErrMountNotExist) {
				return err
			}

			if mntErr := TraceFS.Mount(); mntErr != nil {
				return mntErr
			}
		}
		if strings.TrimSpace(string(b)) == flag {
			log.Printf("%s already %s", event, flag)
			continue
		}
		if err := TraceFS.WriteFile(path, flag, 0666); err != nil {
			log.Printf("error enabling trace event: %v", err)
			continue
		}
	}

	return nil
}

func parseBootstrapEvents() []string {
	if err := ProcFS.Mount(); err != nil && !errors.Is(err, ErrMountExist) {
		log.Fatal(err)
	}

	var events []string

	cmdline, err := ProcFS.ReadFile("cmdline")
	if err != nil {
		log.Fatal(err)
	}

	bootstrapParams := strings.Fields(strings.Trim(string(cmdline), "\\x00[]"))

	log.Printf("bootstrap params: %v", bootstrapParams)
	for _, arg := range bootstrapParams {
		if syscallParam, ok := strings.CutPrefix(arg, "trace_events="); ok {
			for _, event := range strings.Split(syscallParam, ",") {
				if !strings.Contains(event, ":") {
					event = "syscalls:" + event
				}

				events = append(events, event)
			}
		}
	}

	return events
}
