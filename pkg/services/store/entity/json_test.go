package entity

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRawEncoders(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"hello": "world",
		"field": 1.23,
	})
	require.NoError(t, err)

	raw := &Entity{
		GRN: &GRN{
			UID:  "a",
			Kind: "b",
		},
		Version: "c",
		ETag:    "d",
		Body:    body,
		Folder:  "f0",
	}

	expect := `{
		"GRN": {
		  "kind": "b",
		  "UID": "a"
		},
		"version": "c",
		"folder": "f0",
		"access": [
		  {
			"role": "viewer",
			"subject": "dashboards",
			"verb": "read"
		  },
		  {
			"role": "viewer",
			"subject": "playlist",
			"verb": "write"
		  }
		],
		"body": {
		  "field": 1.23,
		  "hello": "world"
		},
		"etag": "d"
	  }`

	b, err := json.MarshalIndent(raw, "", "  ")
	require.NoError(t, err)

	str := string(b)
	fmt.Println(str)
	require.JSONEq(t, expect, str)

	copy := &Entity{}
	err = json.Unmarshal(b, copy)
	require.NoError(t, err)

	b, err = json.MarshalIndent(copy, "", "  ")
	require.NoError(t, err)
	str = string(b)
	require.JSONEq(t, expect, str)
}
