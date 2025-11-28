package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type RequestRule struct {
	Path    string         `yaml:"path" json:"path"`
	Method  string         `yaml:"method" json:"method"`
	Headers map[string]any `yaml:"headers" json:"headers"`
	Body    string         `yaml:"body" json:"body"`
	Params  map[string]string
}

type MockResponse struct {
	Status  uint16         `yaml:"status" json:"status"`
	Headers map[string]any `yaml:"headers" json:"headers"`
	Body    string         `yaml:"body" json:"body"`
}

type ProxyConfig struct {
	Url       string            `yaml:"url" json:"url"`
	Headers   map[string]string `yaml:"headers" json:"headers"`
	TimeoutMs uint64            `yaml:"timeout" json:"timeout"`
}

type Rule struct {
	Request  *RequestRule  `yaml:"request" json:"request"`
	Response *MockResponse `yaml:"response" json:"response"`
	Proxy    *ProxyConfig  `yaml:"proxy" json:"proxy"`
}

func (r Rule) IsProxyStatic() bool {
	paths := strings.SplitSeq(r.Proxy.Url, "/")
	for p := range paths {
		if strings.HasPrefix(p, ":") {
			return false
		}
	}
	return true
}

func (r Rule) IsStatic() bool {
	paths := strings.SplitSeq(r.Request.Path, "/")
	for p := range paths {
		if strings.HasPrefix(p, ":") {
			return false
		}
	}
	return true
}

func (r *Rule) IsMock() bool {
	return r.Response != nil
}

func (r *Rule) IsProxy() bool {
	return r.Proxy != nil
}

type Config struct {
	Rules []Rule
}

func loadFromDirectory(dirPath string) (*Config, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	config := &Config{}

	for _, entry := range entries {
		stat, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if stat.IsDir() {
			continue
		}
		ext := filepath.Ext(filepath.Join(dirPath, entry.Name()))
		if ext != ".yml" && ext != ".yaml" && ext != ".json" {
			continue
		}

		conf, err := loadSingleFile(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse %s/%s: %v", dirPath, entry.Name(), err)
			continue
		}
		for _, c := range conf.Rules {
			config.Rules = append(config.Rules, c)
		}
	}

	return config, nil
}

func loadSingleFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := filepath.Ext(path)
	var rules []Rule

	var decodeErr error

	switch ext {
	case ".yaml", ".yml":
		decodeErr = yaml.NewDecoder(file).Decode(&rules)
	case ".json":
		decodeErr = json.NewDecoder(file).Decode(&rules)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode %q: %s", path, decodeErr.Error())
	}

	return &Config{Rules: rules}, nil
}

func LoadFromFile(path string) (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("%v", err.Error())
	}

	fullPath := filepath.Join(cwd, path)

	file, err := os.OpenFile(fullPath, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("error opening file:", err)
		os.Exit(1)
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %v", err)
	}
	if stat.IsDir() {
		return loadFromDirectory(fullPath)
	} else {
		return loadSingleFile(fullPath)
	}
}
