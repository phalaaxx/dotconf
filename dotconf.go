package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	//"text/template"
	"encoding/json"
)

/* initial command parser */
type CmdParse struct {
	Command     string          `json:"command"`
	Description string          `json:"description"`
	Args        json.RawMessage `json:"args"`
}

/* ArgsInterface is an interface for type assertion */
type ArgsInterface interface {
	Run() error
}

/* ServerConf is the top-level json configuration
   structure provided to the json parser */
type ServerConf struct {
	Commands []CmdParse `json:"configuration"`
}

/* CmdApt is apt related structure */
type CmdApt struct {
	State string   `json:"state"`
	Pkgs  []string `json:"pkgs"`
	Purge bool     `json:"purge"`
}

/* Run apt-get command */
func (c *CmdApt) Run() error {
	if c.State == "present" {
		/* install packages */
		args := append(
			[]string{"-y", "install"},
			c.Pkgs...,
		)
		cmd := exec.Command(
			"/usr/bin/apt-get",
			args...,
		)
		return cmd.Run()
	}
	if c.State == "absent" {
		/* remove or purge */
		purge := ""
		if c.Purge {
			purge = "--purge"
		}
		/* remove packages */
		args := append(
			[]string{"-y", purge, "remove"},
			c.Pkgs...,
		)
		cmd := exec.Command(
			"/usr/bin/apt-get",
			args...,
		)
		return cmd.Run()
	}
	/* unsupported state */
	return fmt.Errorf(
		"Unknown state %s (must be present or absent)",
		c.State,
	)
}

/* user related commands */
type CmdUser struct {
	User    string `json:"user"`
	Present bool   `json:"present"`
	Shell   string `json:"shell"`
	HomeDir string `json:"homedir"`
}

/* Perform user related actions */
func (c *CmdUser) Run() error {
	/* check if user already exists */
	var err error
	var current *user.User
	if current, err = user.Lookup(c.User); err != nil {
		if _, ok := err.(*user.UnknownUserError); !ok {
			return err
		}
		current = nil
	}

	/* check if current user exists */
	if c.Present && current != nil {
		/* user does not exist, create it */
		HomeDir := c.HomeDir
		if len(HomeDir) == 0 {
			HomeDir = fmt.Sprintf("/home/%s", c.User)
		}
		cmd := exec.Command(
			"/usr/sbin/useradd",
			"-m",
			"-d", HomeDir,
			/* TODO: uid, gid, gecos */
		)
		return cmd.Run()
	}

	/* make sure user properties are correct */
	if true {
		/* TODO: get current user shell */
		Shell := c.Shell
		if len(Shell) == 0 {
			Shell = "/usr/bin/zsh"
		}

		HomeDir := c.HomeDir
		if len(HomeDir) == 0 {
			HomeDir = current.HomeDir
		}
		cmd := exec.Command(
			"/usr/sbin/usermod",
			"-s", Shell,
			"-d", HomeDir,
			c.User,
		)
		return cmd.Run()
	}
	return nil
}

func ParseFile(File string) (*ServerConf, error) {
	var file *os.File
	var err error
	if file, err = os.Open(File); err != nil {
		return nil, err
	}
	defer file.Close()

	var conf ServerConf
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&conf); err != nil {
		return nil, err
	}

	/* perform specified tasks */
	for _, cmd := range conf.Commands {
		if err = ParseCommand(cmd); err != nil {
			return nil, err
		}
	}

	return &conf, nil
}

/* Parse configuration file */
func ParseCommand(cmd CmdParse) error {
	var args interface{}
	switch cmd.Command {
	case "apt":
		args = new(CmdApt)
	case "user":
		args = new(CmdUser)
	default:
		return fmt.Errorf(
			"Unknown command: %s", cmd.Command,
		)
	}
	/* parse arguments and run specified action */
	if err := json.Unmarshal(cmd.Args, args); err != nil {
		return err
	}
	/* tell user the current command */
	fmt.Printf(
		"[ %s: %s ]\n",
		cmd.Command,
		cmd.Description,
	)
	/* perform specified command */
	if err := args.(ArgsInterface).Run(); err != nil {
		fmt.Println("  Error:", err)
		return err
	}
	fmt.Println("  OK")
	return nil
}

/* Main program */
func main() {
	_, err := ParseFile("desktop.json")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
