package catalog

import (
	"reflect"
	"strings"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/engine"
)

// Dummy variables for type checking
var intPtr *int64
var floatPtr *float64
var strPtr *string
var strType = reflect.TypeOf("")
var float64Type = reflect.TypeOf(float64(0))

func PopulateArgsWithUsage(args interface{}, u *engine.ConsumptionProfile) {
	if u == nil {
		// nothing to do
		return
	}

	ps := reflect.ValueOf(args)
	s := ps.Elem()
	if s.Kind() != reflect.Struct {
		return
	}

	// Iterate over all fields of the args struct and
	// try to set the values from the usage if it exist.
	for i := 0; i < s.NumField(); i++ {
		// Get the i'th field of the struct
		f := s.Field(i)
		if !f.IsValid() {
			continue
		}
		if !f.CanSet() {
			continue
		}
		// Get the c3x tag data.
		c3xTagSplitted := strings.Split(s.Type().Field(i).Tag.Get("c3x_usage"), ",")
		if len(c3xTagSplitted) < 1 {
			continue
		}
		// Key name for the usage file
		usageKey := c3xTagSplitted[0]
		if usageKey == "" {
			continue
		}

		// Check whether a value for this arg was specified in the usage data.
		if u.Get(usageKey).Exists() {
			// Set the value of the arg to the value specified in the usage data.
			if f.Type() == reflect.TypeOf(floatPtr) {
				f.Set(reflect.ValueOf(u.GetFloat(usageKey)))
				continue
			}

			if f.Type() == reflect.TypeOf(intPtr) {
				f.Set(reflect.ValueOf(u.GetInt(usageKey)))
				continue
			}

			if f.Type() == reflect.TypeOf(strPtr) {
				f.Set(reflect.ValueOf(u.GetString(usageKey)))
				continue
			}

			if f.Type().Elem().Kind() == reflect.Struct {
				if f.IsNil() {
					f.Set(reflect.New(f.Type().Elem()))
				}

				PopulateArgsWithUsage(f.Interface(), &engine.ConsumptionProfile{
					Address:    usageKey,
					Attributes: u.Get(usageKey).Map(),
				})

				continue
			}

			if f.Type() == reflect.MapOf(strType, float64Type) {
				m := make(map[string]float64)
				for k, v := range u.Get(usageKey).Map() {
					m[k] = v.Float()
				}

				f.Set(reflect.ValueOf(m))

				continue
			}

			logging.Logger.Error().Msgf("Unsupported field { UsageKey: %s, Type: %v }", usageKey, f.Type())
		}
	}
}
