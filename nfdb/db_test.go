package nfdb

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"

	"github.com/IMQS/log"
	"github.com/jinzhu/gorm"
	"gotest.tools/assert"
)

const (
	DBHOST     = "localhost"
	DBUSER     = "unittest_user"
	DBPASSWORD = "unittest_password"
	DBNAME     = "nftest"
)

type PolyTable struct {
	BaseModel
	Geometry string
}

func DSN() string {
	return "host=" + DBHOST +
		" user=" + DBUSER +
		" password=" + DBPASSWORD +
		" dbname=" + DBNAME +
		" sslmode=disable"
}

func CreateTestMigrations() []string {
	return []string{`
		CREATE EXTENSION IF NOT EXISTS postgis;

		CREATE TABLE "poly_table"
		(
			"id" BIGSERIAL PRIMARY KEY,
			"geometry" geometry(Polygon,4326)
		)
	`}
}

func CreateTestDB(t *testing.T) *gorm.DB {
	var flags DBConnectFlags
	flags |= DBConnectFlagWipeDB
	log := log.New(log.Stdout)
	db, err := OpenDB(
		log,
		"postgres",
		DSN(),
		MakeMigrations(log, CreateTestMigrations()),
		flags,
	)
	if err != nil {
		t.Fatal(err.Error())
	}

	return db
}

// RemoveWhitespace removes whitespace from strings
func RemoveWhitespace(str string) string {
	var b strings.Builder
	b.Grow(len(str))
	for _, ch := range str {
		if !unicode.IsSpace(ch) {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func TestSQL(t *testing.T) {
	verify := func(in, out string) {
		actual := SQLCleanIDList(in)
		assert.Equal(t, out, actual)
	}

	verify("", "()")
	verify("1", "(1)")
	verify("1,2", "(1,2)")
	verify("a,2", "(2)")
	verify("a", "()")
	verify("1,,3", "(1,3)")
	verify(",", "()")
}

func TestGeom(t *testing.T) {
	db := CreateTestDB(t)
	defer db.Close()

	geoJSON := RemoveWhitespace(`
	{
		"type": "Polygon",
		"coordinates": [
			[
				[0.00013, 0.000131],
				[0.000131, 0.00013],
				[0.000129, 0.00013],
				[0.00013, 0.000131]
			]
		]
	}`)

	err := db.Exec(`
		INSERT INTO "poly_table" (
			"geometry"
		) VALUES (ST_SetSRID(ST_GeomFromGeoJSON(?),4326))
	`, geoJSON).Error
	if err != nil {
		t.Fatal("Failed to insert geometry")
	}

	polys := []PolyTable{}

	err = db.Find(&polys).Error
	if err != nil {
		t.Fatalf("Failed to find a poly in PolyTable: %v", err)
	}
	if len(polys) == 0 {
		t.Fatalf("Failed to find a poly in PolyTable: No Row Returned")
	}

	poly := polys[0]

	// Test getting the geometry column

	columnName, err := GetGeometryColumn(db, &poly)
	if err != nil {
		t.Fatalf("Failed to get the name of the geometry field from the GORM struct 'PolyTable': %v", err)
	}

	if columnName != "Geometry" {
		t.Fatalf("Unexpected geometry field name for GORM struct 'PolyTable'. Expected name should be 'Geometry', got '%v'", columnName)
	}

	// Test getting GetGeoJSON from a model

	geometry, err := GetGeoJSON(db, &poly)
	if err != nil {
		t.Fatalf("Failed while getting GeoJSON: %v", err)
	}

	rawJSON, err := json.Marshal(geometry)
	if err != nil {
		t.Fatalf("Failed while marshaling the geometry received from the db: %v", err)
	}

	if string(rawJSON) != geoJSON {
		t.Fatalf("Unexpected geometry value received from db. Expected value: %v, received value: %v", geoJSON, string(rawJSON))
	}
}
