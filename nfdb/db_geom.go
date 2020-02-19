package nfdb

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
)

var tableGeomMap sync.Map

// GetGeometryColumn returns the name of the geometry column in the GORM struct.
// Returns the columnName and an error
func GetGeometryColumn(db *gorm.DB, model interface{}) (string, error) {
	r := reflect.ValueOf(model).Elem().Type()
	tableName := r.Name()

	if columnName, ok := tableGeomMap.Load(tableName); ok {
		return columnName.(string), nil
	}

	res := struct {
		FGeometryColumn string
	}{}

	err := db.Raw(
		`SELECT "f_geometry_column"
		 FROM "geometry_columns"
		 WHERE "f_table_name" = ?`,
		gorm.TheNamingStrategy.ColumnName(tableName),
	).Scan(&res).Error
	if err != nil {
		if err.Error() == "record not found" {
			return "", errors.New("No geometry field found")
		}
		return "", err
	}

	columnName := ""
	for i := 0; i < r.NumField(); i++ {
		tableColumnName := gorm.TheNamingStrategy.ColumnName(r.Field(i).Name)

		if tableColumnName == res.FGeometryColumn {
			columnName = r.Field(i).Name
			break
		}
	}

	if columnName == "" {
		return "", fmt.Errorf("Could not find matching geometry field in GORM struct")
	}

	tableGeomMap.Store(tableName, columnName)

	return columnName, nil
}

// GetGeoJSON marshals the geometry associated with the model into a GeoJSON
// geometry and returns it.
func GetGeoJSON(db *gorm.DB, model interface{}) (*geojson.Geometry, error) {
	geomColumn, err := GetGeometryColumn(db, model)
	if err != nil {
		return nil, fmt.Errorf("while getting the name of the geometry column of GORM struct: %v", err)
	}

	r := reflect.ValueOf(model).Elem()
	strGeom := r.FieldByName(geomColumn).String()
	rawGeom, err := hex.DecodeString(strGeom)
	if err != nil {
		return nil, fmt.Errorf("while decoding geom string: %v", err)
	}

	geom, err := ewkb.Unmarshal(rawGeom)
	if err != nil {
		return nil, err
	}

	geoJSON, err := geojson.Encode(geom)
	if err != nil {
		return nil, err
	}

	return geoJSON, nil
}
