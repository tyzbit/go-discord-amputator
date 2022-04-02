# go-discord-amputator
Discord Amputator bot rewritten in Go

## Configuration

Set some environment variables before launching, or add a `.env` file.

If database environment variables are provided, the bot will save stats to an external database.
Otherwise, it will save stats to a local sqlite database at `/var/go-discord-amputator/local.db`

| Variable | Value(s) |
|:-|:-|
| ADMINISTRATOR_IDS | IDs of users allowed to use administrator commands |
| DB_DATABASE | Database name for database
| DB_HOST | Hostname for database |
| DB_PASSWORD | Password for database user |
| DB_USER | Username for database user |
| GUESS_AND_CHECK | Whether to ask the API to take guesses at what the canonical URL is |
| LOG_LEVEL | `trace`, `debug`, `info`, `warn`, `error` |
| TOKEN | The Discord token the bot should use |

## Usage

Configure the bot with `!amp config [setting] [value]`. The settings are below:

| Setting | Description | Default |
|:-|:-|-:|
| switch | Enable the bot: `on`, disable the bot: `off` | `on` |
| replyto | Reply to the original message for context, `on` or `off` | `off` |
| embed | Whether to use an embed message or just reply with links (Discord will then auto preview them), `on` or `off` | `on` |
| guess | Whether to guess if the URL is difficult to amputate, `on` or `off` | `on` | 
| maxdepth | The maximum number of links deep to go to find the canonical URL,  any number | `3` |

You can also use `!amp stats` to get amputation stats for your server.