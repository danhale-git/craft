package biomes

import (
	"fmt"
	"strconv"

	"github.com/spf13/viper"
)

func UpdateAllSurfaceMaterials(filePath, value string) error {
	surfaceParamsKey := "minecraft:biome.components.minecraft:surface_parameters"

	v, err := readFileToViper(filePath)
	if err != nil {
		return err
	}

	if v.IsSet(surfaceParamsKey) {
		sp := v.Get(surfaceParamsKey)

		// Walk all surface parameters
		for key, val := range sp.(map[string]interface{}) {
			// Do nothing if not a string value
			if _, ok := val.(string); ok {
				v.Set(fmt.Sprintf("%s.%s", surfaceParamsKey, key), value)
			}
		}
	} else {
		return KeyNotFoundError{
			filePath: filePath,
			key:      surfaceParamsKey,
		}
	}

	err = v.WriteConfig()
	if err != nil {
		return err
	}

	fmt.Println("wrote surface parameters to", filePath)

	return nil
}

// UpdateFile makes a change to a biome file
func UpdateValue(filePath, key string, value interface{}) error {
	v, err := readFileToViper(filePath)
	if err != nil {
		return err
	}

	if !v.IsSet(key) {
		return KeyNotFoundError{filePath: filePath, key: key}
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

	fmt.Println("wrote to", filePath)

	return nil
}

func readFileToViper(filePath string) (*viper.Viper, error) {
	newViper := viper.New()

	newViper.SetConfigType("json")
	newViper.SetConfigFile(filePath)

	if err := newViper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("loading config at %s : %v", filePath, err)
	}

	return newViper, nil
}

type KeyNotFoundError struct {
	filePath, key string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("key '%s' does not exist in file at %s", e.key, e.filePath)
}
