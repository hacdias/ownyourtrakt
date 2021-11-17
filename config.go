package main

import (
	"errors"

	"github.com/spf13/viper"
)

type config struct {
	BaseURL           string
	Port              int
	SessionKey        string
	Database          string
	TraktClientID     string
	TraktClientSecret string
}

func getConfig() (*config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	viper.SetDefault("port", 8050)
	viper.SetDefault("baseUrl", "http://localhost:8050")
	viper.SetDefault("database", "./database.db")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	conf := &config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	if conf.TraktClientID == "" {
		return nil, errors.New("traktClientId must be defined")
	}

	if conf.TraktClientSecret == "" {
		return nil, errors.New("traktClientSecret must be defined")
	}

	if conf.SessionKey == "" {
		return nil, errors.New("sessionKey must be defined")
	}

	return conf, nil
}
