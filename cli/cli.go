package cli

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Cozzytree/apihub/app"
	"github.com/Cozzytree/apihub/config"
	httpserver "github.com/Cozzytree/apihub/http_server"
	"github.com/Cozzytree/apihub/interfaces"
	"github.com/fsnotify/fsnotify"
)

const version = "0.1.1"
const DEFAULT_RATELIMITER_LIMIT = 10
const DEFAULT_RATELIMITER_WINDOW = 10 * time.Second

type ServeConfig struct {
	host             string
	port             uint16
	watch            bool
	max_request_size uint
	request_timeout  time.Duration
	rate_limiter     bool
}

type CLI struct {
}

func Init() *CLI {
	return &CLI{}
}

func (c *CLI) Run(args []string) error {
	if len(args) < 2 {
		c.printUsage()
		return errors.New("")
	}

	command := args[1]
	switch command {
	case "serve":
		c.runServeCmd(args[1:])
	case "validate":

	case "version":
		fmt.Println(version)
	case "-h", "--help":
		c.printUsage()
	default:
		fmt.Println("Unknown command")
		c.printUsage()
		os.Exit(1)
	}

	return nil
}

func (c *CLI) runServeCmd(args []string) {
	serve_conf := ServeConfig{}
	var config_path string = ""

	var i uint = 0
	for i < uint(len(args)) {
		arg := args[i]
		switch arg {
		case "--port", "-p":
			if i+1 >= uint(len(args)) {
				fmt.Println("--port requires a value")
				os.Exit(1)
			}
			port, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("port should be an int")
				os.Exit(1)
			}
			serve_conf.port = uint16(port)
			i += 2
		case "--host", "-h":
			if i+1 >= uint(len(args)) {
				fmt.Println("--host requires a value")
				os.Exit(1)
			}
			serve_conf.host = args[i+1]
			i += 2
		case "--watch", "-w":
			serve_conf.watch = true
			i += 1
		case "-rl":
			serve_conf.rate_limiter = true
			i += 1
		case "--request-timeout":
			if i+1 >= uint(len(args)) {
				fmt.Println("--max-request-size requires a value")
				os.Exit(1)
			}
			timeoutStr := args[i+1]
			duration, err := time.ParseDuration(timeoutStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid request timeout format %q (use values like 30s, 500ms, 2m)\n", timeoutStr)
				os.Exit(1)
			}
			serve_conf.request_timeout = duration
			i += 2
		case "--max-request-size":
			if i+1 >= uint(len(args)) {
				fmt.Println("--max-request-size requires a value")
				os.Exit(1)
			}
			max_req, err := strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("invalid max request size")
				os.Exit(1)
			}
			serve_conf.max_request_size = uint(max_req)
			i += 2
		case "-f", "--file":
			fmt.Println("args", args, i)
			if i+1 >= uint(len(args)) {
				fmt.Println("filepath requires a value")
				os.Exit(1)
			}
			config_path = args[i+1]
			i += 2
		default:
			i += 1
		}
	}

	if config_path == "" {
		config_path = "config.yaml"
	}
	c.startServer(config_path, serve_conf)
}

func (c *CLI) startServer(config_path string, serve_config ServeConfig) {
	log.Println("Starting Server...")
	log.Printf("Config file %s", config_path)
	log.Printf("Host: %s", serve_config.host)
	log.Printf("Port: %d", serve_config.port)
	if serve_config.rate_limiter {
		log.Println("Limiter Enabled")
	}

	app_conf, err := config.LoadFromFile(config_path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	httpSrv := httpserver.CreateHttpServer()

	app_shop := app.Init(httpSrv, *app_conf)

	if serve_config.request_timeout == 0 {
		serve_config.request_timeout = 30 * time.Second
	}

	rateLimit, err := strconv.Atoi(os.Getenv("APIHUB_RATELIMIT"))
	if err != nil {
		rateLimit = DEFAULT_RATELIMITER_LIMIT
	}

	rateLimitWindow, err := time.ParseDuration(os.Getenv("APIHUB_RATEWINDOW"))
	if err != nil {
		rateLimitWindow = DEFAULT_RATELIMITER_WINDOW
	}

	server_config := interfaces.ServerConfig{
		Host:                 serve_config.host,
		Port:                 serve_config.port,
		Max_request_size:     serve_config.max_request_size,
		Request_timeout_ms:   uint64(serve_config.request_timeout.Milliseconds()),
		Rate_limit:           serve_config.rate_limiter,
		Rate_limit_requests:  uint32(rateLimit),
		Rate_limit_window_ms: rateLimitWindow,
	}

	startServer := func() {
		if err := app_shop.Start(server_config); err != nil {
			fmt.Printf("err: %v\n", err)
		}
	}
	// file watcher
	if serve_config.watch {
		log.Printf("Watching: %s", config_path)
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		if err := watcher.Add(config_path); err != nil {
			log.Fatalf("Error watching file: %v", err)
		}

		go func() {
			debounce := time.Now()
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if time.Since(debounce) < 500*time.Millisecond {
						continue
					}
					debounce = time.Now()

					if event.Op&(fsnotify.Write) != 0 {
						// Clear terminal
						fmt.Print("\033[H\033[2J")

						log.Println("Config file modified — reloading server...")

						// reload config and server
						app_shop.Stop()

						newconf, err := config.LoadFromFile(config_path)
						if err != nil {
							log.Printf("Error reloading config: %v", err)
							continue
						}

						httpSrv = httpserver.CreateHttpServer()
						app_shop = app.Init(httpSrv, *newconf)
						startServer()
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					fmt.Println("error", err)
				}
			}
		}()
	}

	startServer()

	// done := make(chan os.Signal, 1)
	// signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// <-done
	// if err := app_shop.Stop(); err != nil {
	// 	fmt.Printf("Error during shutdown: %v\n", err)
	// 	os.Exit(1)
	// }
}

func (c CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("Commands")
	fmt.Println(" serve [config.yaml] start the HTTP server")
	fmt.Println("  -p port -w(watch config file) --max-request-size bytes")
	fmt.Println(" version")
	fmt.Println(" -h or --help")
}
