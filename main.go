package main 

import (
    "fmt"
    "os"
    "log"
    "internal/config"
)

type state struct {
    conf *config.Config
}

type command struct {
    name string
    args []string
}

type commands struct {
    commands map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
    if fx, exists := c.commands[cmd.name]; exists {
        err := fx(s, cmd)
        return err
    } else {
        return fmt.Errorf("Unknown command '%v'\n", cmd.name)
    }
}

func (c *commands) register(name string, f func(*state, command) error) {
    c.commands[name] = f
}

func handlerLogin(s *state, cmd command) error {
    if len(cmd.args) == 0 {
        return fmt.Errorf("The login handler expects the username\n")
    }

    username := cmd.args[0]
    err := config.SetUser(username, s.conf)
    if err != nil {
        return err
    }
    fmt.Printf("The user '%v' has been set\n", username)

    return nil
}

func main() {
	cnf, err := config.Read()
    if err != nil {
        log.Fatalf("%v\n", err)
    }

    sharedState := &state{conf: &cnf}
    args := os.Args
    commands := commands{commands: make(map[string]func(*state, command) error, 16)}
    commands.register("login", handlerLogin)

    if len(args) < 2 {
        log.Fatalf("Not enough arguments\n")
    }
    command := command{name: args[1], args: args[2:]}
    commands.run(sharedState, command)

    // fmt.Printf("%v\n", cnf)
    // fmt.Printf("%v\n", command)
    // fmt.Printf("%v\n", sharedState)
    // fmt.Printf("%v\n", args)

}