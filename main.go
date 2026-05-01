package main 

import (
    "fmt"
    "time"
    "os"
    "log"
    "database/sql"
    "internal/config"
    "internal/database"
    "context"
    "github.com/google/uuid"
    _ "github.com/lib/pq"
)

type state struct {
    conf *config.Config
    db *database.Queries
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

func registerUser(s *state, cmd command) error {
    if len(cmd.args) == 0 {
        return fmt.Errorf("The login handler expects the username\n")
    }

    username := sql.NullString{String: cmd.args[0], Valid: true}
    empty_context := context.Background()
    user_args := database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), 
                                           UpdatedAt: time.Now(), Name: username}
    _, err := s.db.CreateUser(empty_context, user_args)
    if err != nil {
        return err
    }

    // set current user in config
    err = config.SetUser(cmd.args[0], s.conf)
    if err != nil {
        return err
    }

    return nil
}

func handlerLogin(s *state, cmd command) error {
    if len(cmd.args) == 0 {
        return fmt.Errorf("The login handler expects the username\n")
    }

    // check whether user exists in db
    username := sql.NullString{String: cmd.args[0], Valid: true}
    empty_context := context.Background()
    _, err := s.db.GetUser(empty_context, username)
    if err != nil {
        return err
    }

    // set current user in config
    err = config.SetUser(cmd.args[0], s.conf)
    if err != nil {
        return err
    }

    return nil
}

func main() {
	cnf, err := config.Read()
    if err != nil {
        log.Fatalf("%v\n", err)
    }

    // open connection to db
    db, err := sql.Open("postgres", *cnf.DbUrl)
    if err != nil {
        // equivalent to Println followed by a call to os.Exit(1).
        log.Fatalf("Unable to connect to Database %v\n", *cnf.DbUrl);
    }
    dbQueries := database.New(db)

    sharedState := &state{conf: &cnf, db: dbQueries}
    args := os.Args
    commands := commands{commands: make(map[string]func(*state, command) error, 16)}
    commands.register("register", registerUser)
    commands.register("login", handlerLogin)

    if len(args) <= 2 {
        log.Fatalf("Not enough arguments\n")
    }
    command := command{name: args[1], args: args[2:]}
    err = commands.run(sharedState, command)
    if err != nil {
        log.Fatalf("Error in command %v()\n", command)
    }

    // fmt.Printf("%v\n", cnf)
    // fmt.Printf("%v\n", command)
    // fmt.Printf("%v\n", sharedState)
    // fmt.Printf("%v\n", args)

}