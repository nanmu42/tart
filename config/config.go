package config

import "github.com/nanmu42/tart/executor"

type Config struct {
	// Gitlab instance URL, only scheme + host, e.g. https://gitlab.example.com
	GitlabEndpoint string `comment:"Gitlab instance URL, only scheme + host, e.g. https://gitlab.example.com"`
	// runner accessToken
	AccessToken string `comment:"Gitlab Runner access token, which can be obtained by tar register command"`

	// config of executor
	Executor executor.Config `comment:"config of executor"`
}
