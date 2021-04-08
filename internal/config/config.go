package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/chremoas/chremoas-ng/internal/log"

	// Import the remote config driver
	_ "github.com/spf13/viper/remote"
)

func New(filename string) (*Configuration, error) {
	var fileRead, remoteRead bool
	var fileReadErr, remoteReadErr error
	var c Configuration

	log := log.New()

	configNameSpace := os.Getenv("CONFIG_NAMESPACE")
	if configNameSpace == "" {
		configNameSpace = "default"
	}

	configType := os.Getenv("CONFIG_TYPE")
	if configType == "" {
		configType = "yaml"
	}

	viper.SetConfigFile(filename)

	if fileReadErr = viper.ReadInConfig(); fileReadErr == nil {
		log.Info("Successfully read local config file")
		fileRead = true
	}

	if err := viper.BindEnv("consul"); err == nil {
		consul := viper.Get("consul")

		if consul != nil {
			// TODO: This is very rigid. Let's find a better way.
			configPath := fmt.Sprintf("/%s/config", configNameSpace)
			log.Infof("Using %s Config: %s", configType, configPath)
			err := viper.AddRemoteProvider("consul", consul.(string), configPath)
			if err == nil {
				viper.SetConfigType(configType) // because there is no file extension in a stream of bytes, supported extensions are "json", "toml", "yaml", "yml", "properties", "props", "prop"

				if remoteReadErr = viper.ReadRemoteConfig(); remoteReadErr == nil {
					log.Info("Successfully read remote config")
					remoteRead = true
				}
			} else {
				log.Info(err.Error())
			}
		}
	}

	if !fileRead && !remoteRead {
		return nil, fmt.Errorf("unable to read config:\n\tfile=%v\n\tremote=%v|n", fileReadErr, remoteReadErr)
	}

	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	// Let's set a default namespace because a lot of people don't care what it actually is
	if c.Namespace == "" {
		c.Namespace = "default.unset"
	}

	return &c, nil
}
