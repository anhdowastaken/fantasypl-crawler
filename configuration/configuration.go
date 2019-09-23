package configuration

import (
	"sync"

	"github.com/spf13/viper"

	"github.com/anhdowastaken/fantasypl-crawler/logger"
)

// AppConfig structure contains main configuration of the app
type AppConfig struct {
	LogLevel int `mapstructure:"loglevel"`
}

type FplConfig struct {
	Username      string   `mapstructure:"username"`
	Password      string   `mapstructure:"password"`
	LeagueIDs     []string `mapstructure:"leagueids"`
	IgnoreEntries []int    `mapstructure:"ignoreentries"`
}

// ConfigurationManager structure
type ConfigurationManager struct {
	mutex  sync.Mutex
	AppCfg AppConfig
	FplCfg FplConfig
	v      *viper.Viper
}

var instance *ConfigurationManager
var once sync.Once

// New function initialize singleton ConfigurationManager
func New() *ConfigurationManager {
	once.Do(func() {
		instance = &ConfigurationManager{}
		instance.v = viper.New()
	})

	return instance
}

// Load function loads and validates configuration from input file
func (cm *ConfigurationManager) Load(configurationFile string) error {
	log := logger.New()

	var tmp ConfigurationManager
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.v.SetConfigFile(configurationFile)
	cm.v.SetConfigType("toml")
	err := cm.v.ReadInConfig()
	if err != nil {
		log.Fatal.Printf("Fatal error occurs while reading config file: %s \n", err)
		return err
	}

	// Load application config
	err = cm.v.UnmarshalKey("app", &tmp.AppCfg)
	if err != nil {
		log.Critical.Printf("[app] part of config file is not valid: %s \n", err)
		return err
	}

	mi := cm.v.Get("app")
	if mi == nil {
		log.Critical.Printf("[app] part of config file is not valid\n")
		return err
	}

	m := mi.(map[string]interface{})

	if m["loglevel"] == nil {
		tmp.AppCfg.LogLevel = logger.INFO // By default, log level is INFO
	} else {
		logLevel, ok := m["loglevel"].(int64)
		if !ok || (logLevel < logger.FATAL || logLevel > logger.DEBUG) {
			tmp.AppCfg.LogLevel = logger.INFO
		}
	}

	err = cm.v.UnmarshalKey("fpl", &tmp.FplCfg)
	if err != nil {
		log.Critical.Printf("[fpl] part of config file is not valid: %s \n", err)
		return err
	}

	mi = cm.v.Get("fpl")
	if mi == nil {
		log.Critical.Printf("[fpl] part of config file is not valid\n")
		return err
	}

	cm.AppCfg = tmp.AppCfg
	cm.FplCfg = tmp.FplCfg

	log.SetLevel(cm.AppCfg.LogLevel)

	return nil
}
