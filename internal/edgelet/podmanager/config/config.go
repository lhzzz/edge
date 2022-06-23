package config

const (
	defaultProjectPath = "/etc/docker-compose/"
)

type Config struct {
	Project     string //docker-compose need Project
	ProjectPath string
}

type Option interface {
	Apply(*Config)
}

func DefaultConfig() Config {
	return Config{}
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
		c.ProjectPath = defaultProjectPath + project + "/"
	})
}
