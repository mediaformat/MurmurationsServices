package config

import (
	"log"

	env "github.com/caarlos0/env/v6"
)

var Conf = config{}

type config struct {
	Mongo mongoConf
	Nats  natsConf
}

type mongoConf struct {
	USERNAME string `env:"MONGO_USERNAME,required"`
	PASSWORD string `env:"MONGO_PASSWORD,required"`
	HOST     string `env:"MONGO_HOST,required"`
	DBName   string `env:"MONGO_DB_NAME,required"`
}

type natsConf struct {
	ClusterID string `env:"NATS_CLUSTER_ID,required"`
	ClientID  string `env:"NATS_CLIENT_ID,required"`
	URL       string `env:"NATS_URL,required"`
}

func Init() {
	err := env.Parse(&Conf)
	if err != nil {
		log.Fatalf("Failed to decode environment variables: %s", err)
	}
}
