package config

import "path/filepath"

const (
	defaultProject     = "edge"
	defaultProjectPath = "/data/docker-compose/"
)

type Config struct {
	Project     string //docker-compose need Project
	ProjectPath string
	VolumePath  string
	IPAddress   string
}

type Option interface {
	Apply(*Config)
}

func DefaultConfig() Config {
	return Config{
		Project:     defaultProject,
		ProjectPath: defaultProjectPath + defaultProject,
		VolumePath:  defaultProjectPath + defaultProject + "/vol",
	}
}

func (c *Config) EmptyDirRoot() string {
	return filepath.Join(c.VolumePath, "emptydir")
}

func (c *Config) ConfigMapRoot() string {
	return filepath.Join(c.VolumePath, "configmap")
}

func (c *Config) SecretRoot() string {
	return filepath.Join(c.VolumePath, "secret")
}

type funcConfigOption struct {
	f func(co *Config)
}

func (fco *funcConfigOption) Apply(c *Config) {
	fco.f(c)
}

func newFuncConfigOption(f func(c *Config)) *funcConfigOption {
	return &funcConfigOption{
		f: f,
	}
}

func WithProjectName(project string) Option {
	return newFuncConfigOption(func(c *Config) {
		c.Project = project
		c.ProjectPath = defaultProjectPath + project
		c.VolumePath = defaultProjectPath + project + "/vol"
	})
}

func WithIPAddress(address string) Option {
	return newFuncConfigOption(func(c *Config) {
		c.IPAddress = address
	})
}
