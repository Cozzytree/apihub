package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type RequestRule struct {
	Path    string         `yaml:"path"`
	Method  string         `yaml:"method"`
	Headers map[string]any `yaml:"headers"`
	Body    string         `yaml:"body"`
}

type MockResponse struct {
	Status  uint16         `yaml:"status"`
	Headers map[string]any `yaml:"headers"`
	Body    string         `yaml:"body"`
}

type ProxyConfig struct {
	Url       string         `yaml:"url"`
	Headers   map[string]any `yaml:"headers"`
	TimeoutMs uint64         `yaml:"timeout"`
}

type Rule struct {
	Request  *RequestRule  `yaml:"request"`
	Response *MockResponse `yaml:"response"`
	Proxy    *ProxyConfig  `yaml:"proxy"`
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

func loadSingleFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rules []Rule

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&rules); err != nil {
		return nil, err
	}

	return &Config{
		Rules: rules,
	}, nil
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

	} else {
		return loadSingleFile(fullPath)
	}

	return nil, errors.New("invalid path")
}
