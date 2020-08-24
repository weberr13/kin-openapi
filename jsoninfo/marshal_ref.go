package jsoninfo

import (
	"encoding/json"
)

// this is a clone of the definition in openapi3, done so we do not import that package here. todo: move to common consts
const rootObjectDepth = 3

func MarshalRef(value string, otherwise interface{}) ([]byte, error) {
	if len(value) > 0 {
		return json.Marshal(&refProps{
			Ref: value,
		})
	}

	// if we're mashaling a value that is already known as a root component, return a reference to this object instead
	// the value 3 is the depth of the path from the root ( / components / <types> / Object )
	if p, ok := marshalContext.rootComponentRefPaths[otherwise]; ok {
		if len(marshalContext.path) > rootObjectDepth {
			return json.Marshal(&refProps{
				Ref: p,
			})
		}
	}

	// default marshal of value
	return json.Marshal(otherwise)
}

func UnmarshalRef(data []byte, destRef *string, destOtherwise interface{}) error {
	refProps := &refProps{}
	if err := json.Unmarshal(data, refProps); err == nil {
		ref := refProps.Ref
		if len(ref) > 0 {
			*destRef = ref
			return nil
		}
	}
	return json.Unmarshal(data, destOtherwise)
}

type refProps struct {
	Ref string `json:"$ref,omitempty"`
}
