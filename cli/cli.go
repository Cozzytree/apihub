package cli

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Cozzytree/apishop/app"
	"github.com/Cozzytree/apishop/config"
	httpserver "github.com/Cozzytree/apishop/http_server"
	"github.com/Cozzytree/apishop/interfaces"
)

type ServeConfig struct {
	host             string
	port             uint16
	watch            bool
	max_request_size uint
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

	app_conf, err := config.LoadFromFile(config_path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	httpSrv := httpserver.CreateHttpServer()

	app_shop := app.Init(httpSrv, *app_conf)

	server_config := interfaces.ServerConfig{
		Host:             serve_config.host,
		Port:             serve_config.port,
		Max_request_size: serve_config.max_request_size,
	}

	fmt.Printf("Server started in port %d\n", server_config.Port)
	if err := app_shop.Start(server_config); err != nil {
		fmt.Printf("err: %v", err)
		os.Exit(1)
	}
}

func (c CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("Commands")
	fmt.Println(" serve [config.yaml] start the HTTP server")
}
