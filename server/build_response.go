package server

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/default23/protofake/mapper"
)

func buildResponse(
	resp *mapper.Response,
	reqBody map[string]any,
	reqMetadata metadata.MD,
) ([]byte, error) {
	respBody := make(map[string]any)
	if resp.Body != nil {
		respBody = resp.Body
	}

	outjson := "{}"
	for k, v := range respBody {
		var err error
		outjson, err = sjson.Set(outjson, k, getResponseValue(v, reqBody, reqMetadata))
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, "failed to set property "+k+" in output message, verify the registered mappings: "+err.Error())
		}
	}

	return []byte(outjson), nil
}

func getResponseValue(val any, reqBody map[string]any, reqMetadata map[string][]string) any {
	str, ok := val.(string)
	if !ok {
		return val
	}

	if strings.HasPrefix(str, "$req.body.") {
		valuePath := strings.TrimPrefix(str, "$req.body.")

		jsonBody, _ := json.Marshal(reqBody)
		value := gjson.GetBytes(jsonBody, valuePath)
		if !value.Exists() {
			return nil
		}

		return value.Value()
	}
	if strings.HasPrefix(str, "$req.metadata.") {
		valuePath := strings.TrimPrefix(str, "$req.metadata.")

		values, _ := reqMetadata[valuePath]
		return strings.Join(values, " ")
	}

	return val
}
