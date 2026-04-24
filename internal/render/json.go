package render

import (
	"encoding/json"
)

func ToJSON(out Report, opts Options) ([]byte, error) {
	return json.Marshal(out)
}
