# signal-test

Instructions

- `go build` to compile binary. Then run. Program should process ~1 message/sec.

- Change RATELIMIT to something other than 1 (it's in `.env`)

- Find process ID `ps aux | grep signal-test`

- Issue SIGHUP to that process ID (`kill -SIGHUP pid`). Does the program start to faster? It should.

- Keep changing the RATELIMIT and watch as the program reloads without a full restart.

- Issue SIGTERM to the process ID (`kill -SIGTERM pid`). The program should cleanly shutdown.


