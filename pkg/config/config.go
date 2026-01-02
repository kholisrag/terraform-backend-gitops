package config

type Config struct {
	LogLevel    string      `koanf:"logLevel" default:"info"`
	Repo        Repo        `koanf:"repo"`
	Server      Server      `koanf:"server"`
	Build       Build       `koanf:"build"`
	Tracing     Tracing     `koanf:"tracing"`
	Encryptions Encryptions `koanf:"encryptions"`
	Redis       Redis       `koanf:"redis"`
}

type Repo struct {
	RepoLocal  RepoLocal  `koanf:"local"`
	RepoGithub RepoGithub `koanf:"github"`
}

type RepoLocal struct {
	Path string `koanf:"path"`
	Root string `koanf:"root,omitempty"`
}

type RepoGithub struct {
	Enabled       bool         `koanf:"enabled"`
	RemoteURL     string       `koanf:"remoteUrl"`
	Branch        string       `koanf:"branch" default:"main"`
	AuthMethod    string       `koanf:"authMethod" default:"ssh"`
	SSHKeyPath    string       `koanf:"sshKeyPath"`
	Token         string       `koanf:"token"`
	CommitMessage string       `koanf:"commitMessage" default:"chore: update terraform state [automated]"`
	AutoPush      bool         `koanf:"autoPush" default:"true"`
	Author        CommitAuthor `koanf:"author"`
	RetryAttempts int          `koanf:"retryAttempts" default:"3"`
	RetryDelay    int          `koanf:"retryDelay" default:"5"`
}

type CommitAuthor struct {
	Name  string `koanf:"name" default:"Terraform Backend GitOps"`
	Email string `koanf:"email" default:"terraform-backend@gitops.local"`
}

type Build struct {
	Version    string `koanf:"version,omitempty"`
	CommitHash string `koanf:"commitHash,omitempty"`
	BuildTime  string `koanf:"buildTime,omitempty"`
}

type Server struct {
	Mode    string `koanf:"mode" default:"release"`
	Address string `koanf:"address" default:"0.0.0.0:20002"`
}

type Tracing struct {
	Enabled    bool    `koanf:"enabled" default:"true"`
	Provider   string  `koanf:"provider" default:"stdout"`
	SampleRate float64 `koanf:"sampleRate" default:"0.1"`
	OTLP       OTLP    `koanf:"otlp"`
}

type OTLP struct {
	Endpoint string `koanf:"endpoint" default:"0.0.0.0:4317"`
}

type Encryptions struct {
	Mode string `koanf:"mode" default:"age"`
	Age  Age    `koanf:"age"`
}

type Age struct {
	Recipient         string `koanf:"recipient" default:""`
	AgePrivateKeyPath string `koanf:"keys" default:""`
}

type Redis struct {
	Addresses []string `koanf:"addresses"`
}

func NewDefaultConfig() *Config {
	return &Config{}
}
