package config

import (
	"spider/client"
	"spider/module"
	"spider/schedluer"
)

func NewDefaultConfig(domain []string, process []module.ProcessItem, parser []module.ParserResponse) *schedluer.Config {
	moduleConfig := schedluer.ConfigOfModule{
		NumberOfDownload: 2,
		NumberOfPipeline: 1,
		NumberOfAnalyzer: 1,
		Domain:           domain,
	}

	config := schedluer.Config{
		ConfigOfModule: moduleConfig,
		Client:         client.GenHttpClient(),
		Processors:     process,
		ParserResponse: parser,
	}

	return &config
}
