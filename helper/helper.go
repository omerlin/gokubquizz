package helper

import (
	"os"
	"strings"
)

// Transform a Yaml to a generic struct using generics
func Convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = Convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = Convert(v)
		}
	}
	return i
}

// Transform list of environment variable to a map of environment variable
func getEnvironnement() map[string]string {
	items := make(map[string]string)
	for _, item := range os.Environ() {
		vals := strings.SplitN(item, "=", 2)
		items[vals[0]] = vals[1]
	}
	return items
}

func FilteredEnvValues(filterList []string) map[string]string {
	// Current env in a form of a list of key=value
	envValues := getEnvironnement()
	// Filter some keys
	for _, key := range filterList {
		delete(envValues, key)
	}
	return envValues
}
