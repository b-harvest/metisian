package main

import (
	"flag"
	"fmt"
	"github.com/b-harvest/metisian/log"
	"github.com/b-harvest/metisian/metis"
	"github.com/rs/zerolog"
	"os"
)

var (
	cfg *metis.Config
)

func init() {

	var (
		configFilePath  string
		configFileToken string
		logLevel        string

		EnvConfigFilePath  = "CONFIG_FILE_PATH"
		EnvConfigFileToken = "CONFIG_TOKEN"

		DefaultConfigFilePath = "config.toml"
	)

	flag.StringVar(&configFilePath, "config", DefaultConfigFilePath,
		fmt.Sprintf("configuration toml file path you'll use. and also you can set this value through env %s.\n"+
			"If both set, env value will be used.", EnvConfigFilePath))
	flag.StringVar(&configFileToken, "config-token", "",
		fmt.Sprintf("If you set this, it'll be use as Authorization Header with `Bearer $CONFIG_TOKEN`.\n"+
			"you could also set this value with env %s.\n"+
			"If both set, env value will be used.", EnvConfigFileToken))
	flag.StringVar(&logLevel, "log-level", "info", "log level you would show. (debug, info, warn, error...)")

	flag.Parse()

	if configFilePath == DefaultConfigFilePath && os.Getenv(EnvConfigFilePath) != "" {
		configFilePath = os.Getenv(EnvConfigFilePath)
	}
	if configFileToken == "" && os.Getenv(EnvConfigFileToken) != "" {
		configFileToken = os.Getenv(EnvConfigFileToken)
	}

	l, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(l)

	cfg, err = metis.LoadConfig(configFilePath, configFileToken)
	if err != nil {
		panic(err)
	}
}

func main() {

	client, err := metis.NewClient(cfg)
	if err != nil {
		panic(err)
	}

	defer client.Cancel()

	go client.Run()

	var seqAddrsMsg string
	for n, seq := range client.Sequencers {
		if n == metis.MetisianName {
			continue
		}
		seqAddrsMsg = fmt.Sprintf("%s[%s] %s\n", seqAddrsMsg, n, seq.Address)
	}
	log.Info(fmt.Sprintf("Starting monitor metis sequencers...\n%s", seqAddrsMsg))

	<-client.Ctx.Done()

}
