# go-discord-amputator
Discord Amputator bot rewritten in Go

## Configuration

Set some environment variables before launching, or add a `.env` file.

If database environment variables are provided, the bot will save stats to the database.

| Variable | Value(s) |
|:-|:-|
| ADMINISTRATOR_IDS | IDs of users allowed to use administrator commands |
| AUTOMATICALLY_AMPUTATE | If set to any value, determines whether the bot automatically replies to links it thinks are amputatable. |
| BOT_ID | Unique (among bots connecting to the database) ID for the bot (needs to be a number) |
| DB_DATABASE | Database name for database
| DB_HOST | Hostname for database |
| DB_PASSWORD | Password for database user |
| DB_USER | Username for database user |
| GUESS_AND_CHECK | Whether to ask the API to take guesses at what the canonical URL is |
| LOG_LEVEL | `debug`, `info`, `warn`, `error` |
| MAX_DEPTH | How many pages deep to go to find the canonical URL |
| TOKEN | The Discord token the bot should use |
