package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type tinkConfig struct {
	syslogHost         string
	insecureRegistries []string
	httpProxy          string
	httpsProxy         string
	noProxy            string
}

type dockerConfig struct {
	Debug              bool              `json:"debug"`
	LogDriver          string            `json:"log-driver,omitempty"`
	LogOpts            map[string]string `json:"log-opts,omitempty"`
	InsecureRegistries []string          `json:"insecure-registries,omitempty"`
}

func main() {
	fmt.Println("Starting Tink-Docker")
	go rebootWatch()

         fmt.Println("Make /dev/null writeable for all users!")
         cmd := exec.Command("chmod", "666", "/dev/null")
         cmd.Stdout = os.Stdout
         cmd.Stderr = os.Stderr
         err := cmd.Run()
         if err != nil {
             panic(err)
         }


	// Parse the cmdline in order to find the urls for the repository and path to the cert
	content, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		panic(err)
	}
	cmdLines := strings.Split(string(content), " ")
	cfg := parseCmdLine(cmdLines)

	fmt.Println("Starting the Docker Engine")

	d := dockerConfig{
		Debug:     true,
		LogDriver: "syslog",
		LogOpts: map[string]string{
			"syslog-address": fmt.Sprintf("udp://%v:5140", cfg.syslogHost),
		},
		InsecureRegistries: cfg.insecureRegistries,
	}
	path := "/etc/docker"
	// Create the directory for the docker config
	err = os.MkdirAll(path, os.ModeDir)
	if err != nil {
		panic(err)
	}
	if err := d.writeToDisk(filepath.Join(path, "daemon.json")); err != nil {
		panic(fmt.Sprintf("Failed to write docker config: %v", err))
	}

	// Build the command, and execute
	cmd = exec.Command("/usr/local/bin/docker-init", "/usr/local/bin/dockerd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	myEnvs := make([]string, 0, 3)
	myEnvs = append(myEnvs, fmt.Sprintf("HTTP_PROXY=%s", cfg.httpProxy))
	myEnvs = append(myEnvs, fmt.Sprintf("HTTPS_PROXY=%s", cfg.httpsProxy))
	myEnvs = append(myEnvs, fmt.Sprintf("NO_PROXY=%s", cfg.noProxy))

	cmd.Env = append(os.Environ(), myEnvs...)

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

// writeToDisk writes the dockerConfig to loc.
func (d dockerConfig) writeToDisk(loc string) error {
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("unable to marshal docker config: %w", err)
	}
	if err := os.WriteFile(loc, b, 0o600); err != nil {
		return fmt.Errorf("error writing daemon.json: %w", err)
	}

	return nil
}

// parseCmdLine will parse the command line.
func parseCmdLine(cmdLines []string) (cfg tinkConfig) {
	for i := range cmdLines {
		cmdLine := strings.Split(cmdLines[i], "=")
		if len(cmdLine) == 0 {
			continue
		}

		switch cmd := cmdLine[0]; cmd {
		case "syslog_host":
			cfg.syslogHost = cmdLine[1]
		case "insecure_registries":
			cfg.insecureRegistries = strings.Split(cmdLine[1], ",")
		case "HTTP_PROXY":
			cfg.httpProxy = cmdLine[1]
		case "HTTPS_PROXY":
			cfg.httpsProxy = cmdLine[1]
		case "NO_PROXY":
			cfg.noProxy = cmdLine[1]
		}
	}
	return cfg
}

func rebootWatch() {
	fmt.Println("Starting Reboot Watcher")

	// Forever loop
	for {
		if fileExists("/worker/reboot") {
			cmd := exec.Command("/sbin/reboot")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			// wait 3 sec to do actual reboot before workflow send back success status
			time.Sleep(3*time.Second)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
			break
		}
		// Wait one second before looking for file
		time.Sleep(time.Second)
	}
	fmt.Println("Rebooting")
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
