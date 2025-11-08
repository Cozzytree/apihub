# apihub

**apihub** is a lightweight, config-driven API hub that lets you define mock responses or proxy routes using a simple YAML configuration file.
It’s designed for quick testing, simulating APIs, or routing requests without writing server code.

---

## Features

-  **Config-driven** — define routes in YAML or JSON
-  **Proxy support** — forward requests to external APIs
-  **Mock responses** — serve static data instantly
-  **Path parameters** — like `/users/:id`
-  **Validate configs** — before serving
-  **Extensible CLI** — add your own commands easily

---

## Installation

```bash
git clone https://github.com/<your-username>/apihub.git
cd apihub
go build -o apihub
```
## or
```bash
go install github.com/<your-username>/apihub@latest
```
[Examples Folder](https://github.com/cozzytree/apihub/examples)

## Usage
```bash
apihub [options]

Commands:
  serve -f [config file] -p [port] -w [watch config file] --max-request-size [bytes]
  version
  validate
