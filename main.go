package main 

import (
    "fmt"
    "time"
    "os"
    "log"
    "strconv"
    "context"
    "database/sql"
    "github.com/google/uuid"
    _ "github.com/lib/pq"
    "github.com/InfinitasFish/blog_gator/internal/config"
    "github.com/InfinitasFish/blog_gator/internal/database"
    "github.com/InfinitasFish/blog_gator/internal/rss"
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
                                           UpdatedAt: time.Now(), UserName: username}
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

func handlerLogin(s *state, cmd command,) error {
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
    if len(cmd.args) != 1 {
        return fmt.Errorf("The Aggregation handler expects the interval between requests.\nFor example '1s', '1m', '1h'.\n")
    }

    fetch_interval, err := time.ParseDuration(cmd.args[0])
    if err != nil {
        return err
    }
    fmt.Printf("Collecting feeds every %v\n", fetch_interval)

    ticker := time.NewTicker(fetch_interval)
    for ; ; <-ticker.C {
        err = scrapeFeeds(s)
        if err != nil {
            // not ending endless loop
            log.Println(err)
        }
    }

    return nil
}

func handlerAddFeed(s *state, cmd command, user_db database.User) error {
    if len(cmd.args) != 2 {
        return fmt.Errorf("The AddFeed handler expects the name and url\n")
    }

    empty_context := context.Background()
    feed_name := sql.NullString{String: cmd.args[0], Valid: true}
    feed_url := cmd.args[1]
    feed_id := uuid.New()
    feed_args := database.AddFeedParams{ID: feed_id, CreatedAt: time.Now(), UpdatedAt: time.Now(),
                                    FeedName: feed_name, FeedUrl: feed_url, UserID: user_db.ID}
    _, err := s.db.AddFeed(empty_context, feed_args)
    if err != nil {
        return err
    }

    // automatically add feed follow for current user
    feed_follow_args := database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
                                                    UserID: user_db.ID, FeedID: feed_id}
    _, err = s.db.CreateFeedFollow(empty_context, feed_follow_args)
    if err != nil {
        return err
    }

    return nil
}

func handlerListFeeds(s *state, cmd command) error {
    empty_context := context.Background()
    feeds_rows, err := s.db.ListFeeds(empty_context)
    if err != nil {
        return err
    }

    for _, feed := range feeds_rows {
        feed_name := feed.FeedName.String
        user_name := feed.UserName.String
        feed_url := feed.FeedUrl
        fmt.Printf("Feed: %v, User: %v (url: %v)\n", feed_name, user_name, feed_url)
    }

    return nil
}

func handlerFollowFeed(s *state, cmd command, user_db database.User) error {
    if len(cmd.args) != 1 {
        return fmt.Errorf("The FollowFeed handler expects the feed url\n")
    }

    empty_context := context.Background()
    feed_url := cmd.args[0]
    feed_row, err := s.db.GetFeedByUrl(empty_context, feed_url)
    if err != nil {
        return err
    }

    feed_follow_args := database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
                                            UserID: user_db.ID, FeedID: feed_row.ID}
    _, err = s.db.CreateFeedFollow(empty_context, feed_follow_args)
    if err != nil {
        return err
    }
    fmt.Printf("User '%v' followed a Feed '%v' (url: %v)\n", user_db.UserName.String, feed_row.FeedName.String, feed_row.FeedUrl)

    return nil
}

func handlerUserFollowing(s *state, cmd command, user_db database.User) error {
    if (len(cmd.args) != 1) && (len(cmd.args) != 0) {
        return fmt.Errorf("The UserFollowing handler expects the user's name\n")
    }

    empty_context := context.Background()
    var user_name sql.NullString
    // current user
    if len(cmd.args) == 0 {
        user_name = user_db.UserName
    // given cmd user
    } else {
        user_name = sql.NullString{String: cmd.args[0], Valid: true}
    }

    user_follows, err := s.db.GetFeedFollowsForUser(empty_context, user_name)
    if err != nil {
        return err
    }

    fmt.Printf("User '%v' follows feeds:\n", user_name.String)
    for _, follow := range user_follows {
        fmt.Printf("  - %v (url: %v)\n", follow.FeedName.String, follow.FeedUrl)
    }
    return nil
}

func handlerUnfollowFeed(s *state, cmd command, user_db database.User) error {
    if len(cmd.args) != 1 {
        return fmt.Errorf("The UnfollowFeed handler expects the feed's url\n")
    }
    
    empty_context := context.Background()
    feed_db, err := s.db.GetFeedByUrl(empty_context, cmd.args[0])
    if err != nil {
        return err
    }

    delete_args := database.DeleteFeedFollowParams{UserID: user_db.ID, FeedID: feed_db.ID}
    err = s.db.DeleteFeedFollow(empty_context, delete_args)
    if err != nil {
        return err
    }

    return nil
}

func handlerBrowsePosts(s *state, cmd command, user_db database.User) error {
    var posts_limit int
    var err error
    if len(cmd.args) != 1 {
        posts_limit = 2
    } else {
        posts_limit, err = strconv.Atoi(cmd.args[0])
        if err != nil {
            return err
        }
    }

    empty_context := context.Background()
    get_posts_args := database.GetLatestPostsForUserParams{UserID: user_db.ID, Limit: int32(posts_limit)}
    posts_rows, err := s.db.GetLatestPostsForUser(empty_context, get_posts_args)
    if err != nil {
        return err
    }

    for _, post_row := range posts_rows {
        fmt.Printf("%v (%v):\n  - %v\n", post_row.PostTitle, post_row.PostUrl, post_row.Description.String)
    }

    return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
    return func(s *state, cmd command) error {
        user_name := sql.NullString{String: *s.conf.UserName, Valid: true}
        empty_context := context.Background()
        user_db, err := s.db.GetUser(empty_context, user_name)
        if err != nil {
            return err
        }
        return handler(s, cmd, user_db)
    }
}

func scrapeFeeds(s *state) error {
    // find oldest or null fetched feed -> fetch feed -> mark feed as fetched -> add rss' posts to db
    empty_context := context.Background()
	feed_db, err := s.db.GetNextFeedToFetch(empty_context)
    if err != nil {
        return err
    }

    rss_feed, err := rss.FetchFeed(empty_context, feed_db.FeedUrl)
    if err != nil {
        return err
    }

    fetch_time := sql.NullTime{Time: time.Now(), Valid: true}
    fetch_args := database.MarkFeedFetchedParams{ID: feed_db.ID, LastFetchedAt: fetch_time}
    err = s.db.MarkFeedFetched(empty_context, fetch_args)
    if err != nil {
        return err
    }

    fmt.Printf("Scraped feed %v\n", rss_feed.Channel.Title)
    for _, item := range rss_feed.Channel.Item {
    
        var published_time sql.NullTime
        pub_time, err := time.Parse("2020-03-15 07:31:42.23", item.PubDate)
        if err != nil {
            published_time = sql.NullTime{Time: pub_time, Valid: false}
        } else {
            published_time = sql.NullTime{Time: pub_time, Valid: true}
        }
        
        var description sql.NullString
        if len(item.Description) > 0 {
            description = sql.NullString{String: item.Description, Valid: true}
        } else {
            description = sql.NullString{String: item.Description, Valid: false}
        }

        post_args := database.CreatePostParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
                                            PostTitle: item.Title, PostUrl: item.Link, 
                                            Description: description,
                                            PublishedAt: published_time,
                                            FeedID: feed_db.ID}
        _, err = s.db.CreatePost(empty_context, post_args)
        if err != nil {
            return err
        }
        fmt.Printf("Added post %v to db\n", item.Title)
    }

    return nil
}

// say wallahi bro
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
    commands.register("feeds", handlerListFeeds)
    commands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
    commands.register("follow", middlewareLoggedIn(handlerFollowFeed))
    commands.register("following", middlewareLoggedIn(handlerUserFollowing))
    commands.register("unfollow", middlewareLoggedIn(handlerUnfollowFeed))
    commands.register("browse", middlewareLoggedIn(handlerBrowsePosts))

    if len(args) <= 1 {
        log.Fatalf("Not enough arguments\n")
    }
    command := command{name: args[1], args: args[2:]}
    err = commands.run(sharedState, command)
    if err != nil {
        log.Fatalf("Error in command %v(): %v\n", command, err)
    }
}
