package geojson

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/store"
)

func GetEntityKindInfo() models.EntityKindInfo {
	return models.EntityKindInfo{
		ID:            models.StandardKindGeoJSON,
		Name:          "GeoJSON",
		Description:   "JSON formatted spatial data",
		FileExtension: ".geojson",
		MimeType:      "application/json",
	}
}

// Very basic geojson validator
func GetEntitySummaryBuilder() models.EntitySummaryBuilder {
	return func(ctx context.Context, uid string, body []byte) (*models.EntitySummary, []byte, error) {
		var geojson map[string]interface{}
		err := json.Unmarshal(body, &geojson)
		if err != nil {
			return nil, nil, err
		}

		ftype, ok := geojson["type"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("missing type")
		}

		body, err = json.Marshal(geojson)
		if err != nil {
			return nil, nil, err
		}

		summary := &models.EntitySummary{
			Kind: models.StandardKindGeoJSON,
			Name: store.GuessNameFromUID(uid),
			UID:  uid,
			Fields: map[string]interface{}{
				"type": ftype,
			},
		}

		if ftype == "FeatureCollection" {
			features, ok := geojson["features"].([]interface{})
			if ok {
				summary.Fields["count"] = len(features)
			}
		}

		return summary, body, nil
	}
}
