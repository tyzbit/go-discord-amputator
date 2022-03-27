# go-discord-amputator
Discord Amputator bot rewritten in Go

## Configuration

Set some environment variables before launching, or add a `.env` file

| Variable | Value(s) |
|:-|:-|
ADMINISTRATOR_ROLE | ID of Bot Administrator Role |
AUTOMATICALLY_AMPUTATE | `true` or `false`, determines whether the bot automatically replies to links it thinks are amputatable. |
GUESS_AND_CHECK | Whether to ask the API to take guesses at what the canonical URL is |
LOG_LEVEL | `debug`, `info`, `warn`, `error` |
MAX_DEPTH | How many pages deep to go to find the canonical URL |
TOKEN | The Discord token the bot should use |
