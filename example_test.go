package main

import (
	"fmt"
	"os"
)

func Example_usage() {
	os.Args = append(os.Args, "-h")
	fmt.Println()
	main()

	// Output:
	// Usage:
	//   LOGGER_PRETTY               bool           default false
	//   LOGGER_LEVEL                string         default info
	//   LOGGER_CALLER               bool           default false
	//   LOGGER_TIMESTAMP            bool           default false
	//   AWS_ENDPOINT_URL            string         default <empty>
	//   AWS_S3_USE_PATH_STYLE       bool           default false
	//   REDIS_INIT_ADDRESS          []string       required
	//   REDIS_USERNAME              string         required
	//   REDIS_PASSWORD              string         required
	//   REDIS_SELECT_DB             int            required
	//   REDIS_DISABLE_CACHE         bool           default false
	//   STORAGE_BUCKET              string         required
	//   SERVER_ADDRESS              string         default :8080
	//   SERVER_SHUTDOWN_TIMEOUT     time.Duration  default 30s
	//   SERVER_READ_HEADER_TIMEOUT  time.Duration  default 5s
	//   HEALTHCHECK_CACHE_DURATION  time.Duration  default 1s
	//   HEALTHCHECK_TIMEOUT         time.Duration  default 10s
}
