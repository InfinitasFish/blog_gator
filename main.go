package main 

import (
    "fmt"
    "time"
    "os"
    "log"
    "context"
    "database/sql"
    "github.com/google/uuid"
    _ "github.com/lib/pq"
    "internal/config"
    "internal/database"
    "internal/rss"
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

func handlerRegisterUser(s *state, cmd command) error {
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

func handlerResetUsers(s* state, cmd command) error {
    empty_context := context.Background()
    err := s.db.ResetTable(empty_context)
    if err != nil {
        return err
    }
    return nil
}

func handlerListUsers(s* state, cmd command) error {
    empty_context := context.Background()
    users, err := s.db.ListUsers(empty_context)
    if err != nil {
        return err
    }

    currentUser := *s.conf.UserName
    for _, sqlUsr := range users {
        if (sqlUsr.String == currentUser) {
            fmt.Printf("* %v (current)\n", sqlUsr.String)
        } else {
            fmt.Printf("* %v\n", sqlUsr.String)
        }
    }
    return nil
}

func handlerAggregation(s *state, cmd command) error {
    empty_context := context.Background()
    feedUrl := "https://www.wagslane.dev/index.xml"
	
	rssFeed, err := rss.FetchFeed(empty_context, feedUrl)
    if err != nil {
        return err
    }

    fmt.Println(rssFeed)
    return nil
}

func handlerAddFeed(s *state, cmd command) error {
    if len(cmd.args) != 2 {
        return fmt.Errorf("The AddFeed handler expects the name and url\n")
    }

    empty_context := context.Background()
    feed_name := sql.NullString{String: cmd.args[0], Valid: true}
    user_name := sql.NullString{String: *s.conf.UserName, Valid: true}
    feed_url := cmd.args[1]
    user, err := s.db.GetUser(empty_context, user_name)
    if err != nil {
        return err
    }
    user_id := user.ID
    
    feed_args := database.AddFeedParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
                                    Name: feed_name, FeedUrl: feed_url, UserID: user_id}
    _, err = s.db.AddFeed(empty_context, feed_args)
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
    commands.register("login", handlerLogin)
    commands.register("register", handlerRegisterUser)
    commands.register("reset", handlerResetUsers)
    commands.register("users", handlerListUsers)
    commands.register("agg", handlerAggregation)
    commands.register("addfeed", handlerAddFeed)

    if len(args) <= 1 {
        log.Fatalf("Not enough arguments\n")
    }
    command := command{name: args[1], args: args[2:]}
    err = commands.run(sharedState, command)
    if err != nil {
        log.Fatalf("Error in command %v(): %v\n", command, err)
    }
}
