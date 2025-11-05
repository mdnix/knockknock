package config

type Config struct {
	BinaryName      string
	InstallationDir string
	Repo            string
	Version         string

	Auth *AuthConfig
}

type AuthConfig struct {
	Username string
	Password string
	Token    string
}

func New(binaryName string) *Config {
	return &Config{
		BinaryName:      binaryName,
		InstallationDir: "/opt",
	}
}

func (c *Config) WithRepo(repo string) *Config {
	c.Repo = repo
	return c
}

func (c *Config) WithVersion(version string) *Config {
	c.Version = version
	return c
}

func (c *Config) WithAuth(auth *AuthConfig) *Config {
	c.Auth = auth
	return c
}

func (c *Config) WithInstallationDir(dir string) *Config {
	c.InstallationDir = dir
	return c
}
