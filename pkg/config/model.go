package config

type Config struct {
	AppEnv    string          `mapstructure:"app_env"`
	LogLevel  string          `mapstructure:"log_level"`
	Server    ServerConfig    `mapstructure:"server"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Auth      AuthConfig      `mapstructure:"auth"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Services  []ServiceConfig `mapstructure:"services"`
}

type ServerConfig struct {
	Port    int `mapstructure:"port"`
	Timeout int `mapstructure:"timeout"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AuthConfig struct {
	JWTSecret                  string `mapstructure:"jwt_secret"`
	AccessTokenExpirationTime  int    `mapstructure:"access_token_expiration_time"`
	RefreshTokenExpirationTime int    `mapstructure:"refresh_token_expiration_time"`
}

type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	Burst             int `mapstructure:"burst"`
}

type ServiceConfig struct {
	Name     string   `mapstructure:"name"`
	BasePath string   `mapstructure:"base_path"`
	Target   string   `mapstructure:"target"`
	Methods  []string `mapstructure:"methods"`
	SkipAuth bool     `mapstructure:"skip_auth"`
}
