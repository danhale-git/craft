package biomes

import (
	"fmt"
	"strconv"

	"github.com/spf13/viper"
)

// UpdateFile makes a change to a biome file
func UpdateFile(filePath, key string, value interface{}) error {
	v, err := readFileToViper(filePath)
	if err != nil {
		return err
	}

	if !v.IsSet(key) {
		return fmt.Errorf("key '%s' does not exist in file at %s", key, filePath)
	}

	_, ok := v.Get(key).(float64)

	if ok {
		// Value is number
		f, err := strconv.ParseFloat(value.(string), 64)
		if err != nil {
			return fmt.Errorf("value of setting '%s' is a number but given value of '%s' is not", key, value)
		}

		v.Set(key, f)
	} else {
		// Value is string
		v.Set(key, value)
	}

	err = v.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

func readFileToViper(filePath string) (*viper.Viper, error) {
	newViper := viper.New()

	newViper.SetConfigType("json")
	newViper.SetConfigFile(filePath)

	if err := newViper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("loading config at %s : %v\n", filePath, err)
	}

	return newViper, nil
}
