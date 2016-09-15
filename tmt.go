package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

var flagConfFile string

type windowData struct {
	Name string
	Path string
}

func (w windowData) Create(SessionName string) {
	cmd := exec.Command("tmux", "list-window", "-t", SessionName)
	outPipe, _ := cmd.StdoutPipe()
	outputStream := bufio.NewScanner(outPipe)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Could not retrieve window list from session '%s'\n", SessionName)
	}
	exists := false
	for outputStream.Scan() {
		if strings.Contains(outputStream.Text(), w.Name) {
			exists = true
			break
		}
	}
	if len(w.Path) == 0 {
		w.Path = os.Getenv("HOME")
	} else {
		w.Path = strings.Replace(w.Path, "$HOME", os.Getenv("HOME"), -1)
	}
	if exists == false {
		fmt.Printf("--> Creating window '%s'\n", w.Name)
		cmd = exec.Command("tmux", "new-window", "-n", w.Name, "-t", SessionName, "-c", w.Path)
		err = cmd.Run()
		if err != nil {
			fmt.Printf("--> Could not create window '%s'\n", w.Name)
		}
	} else {
		fmt.Printf("--> Window '%s' already exists\n", w.Name)
	}
	cmd.Wait()
}

type sessionData struct {
	Name    string
	Path    string
	Windows []windowData
}

func (s sessionData) Create() bool {
	cmd := exec.Command("tmux", "has-session", "-t", s.Name)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("-> Session '%s' is not running yet... creating\n", s.Name)
		if len(s.Path) == 0 {
			s.Path = os.Getenv("HOME")
		} else {
			s.Path = strings.Replace(s.Path, "$HOME", os.Getenv("HOME"), -1)
		}
		cmd = exec.Command("tmux", "new-session", "-d", "-s", s.Name, "-c", s.Path)
		err = cmd.Run()
		if err != nil {
			fmt.Printf("-> Could not create session '%s', error %s", s.Name, err)
		}
		return true
	} else {
		fmt.Printf("-> Session '%s' already running\n", s.Name)
		return false
	}
}

func (s sessionData) RemoveWindow(WindowName string) {
	cmd := exec.Command("tmux", "kill-window", "-t", s.Name+":"+WindowName)
	cmd.Run()
}

type tmtConf struct {
	SessionData []sessionData
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -c CONF_FILE\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&flagConfFile, "c", path.Join(os.Getenv("HOME"), "tmt.json"), "TMT configuration file (JSON format)")
}

func main() {
	flag.Parse()
	fmt.Printf("Using config file %s\n", flagConfFile)
	file, err := os.Open(flagConfFile)
	if err != nil {
		fmt.Printf("Could not open config file\n")
		return
	}
	decoder := json.NewDecoder(file)
	configuration := tmtConf{}
	err = decoder.Decode(&configuration)
	if err != nil {
		fmt.Printf("Could not parse session data: %s\n", err)
	}
	for _, newSession := range configuration.SessionData {
		fresh := newSession.Create()
		singleWindow := windowData{"default", newSession.Path}
		if len(newSession.Windows) == 0 {
			newSession.Windows = append(newSession.Windows, singleWindow)
		}
		for _, newWindow := range newSession.Windows {
			newWindow.Create(newSession.Name)
		}
		// Remove the default one
		if fresh == true {
			newSession.RemoveWindow("0")
		}
	}
}
