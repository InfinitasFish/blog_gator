CLI tool to scrape standardized [RSS](https://www.rssboard.org/rss-specification#whatIsRss) feeds and recieve their most recent posts.


# Installation

Install PostgreSQL server, create a .gatorconfig.json in HOME directory with these keys (your db url and any name):
```json
{
	"db_url": "postgres://postgres:7952@localhost:5432/gator?sslmode=disable",
	"current_user_name": "alice"
}
```
Then do ```go install github.com/InfinitasFish/blog_gator@v1.0.0```, after that you can use ```blog_gator``` as executable.


# Some commands:

- register (name) - register a new user
- login (name) - set current user (should be registered)
- addfeed (name, url) - add feed to scrape posts from
- follow (url) - subsribe current user to added feed
- agg (interval like '10s', '1m') - start scraping of posts from followed feeds
- browse [limit] - view last 'limit' scraped recent posts


# Tech

For saving users, feeds, posts and connections between them, project utilises:
- PostgreSQL database, 
- Golang Postgre driver [pq](https://pkg.go.dev/github.com/lib/pq), 
- [goose](https://github.com/pressly/goose) DB migrations, 
- [sqlc](https://docs.sqlc.dev/en/latest/index.html) (Sql-queries to Go-code) generation
