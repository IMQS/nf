package nfdb

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/migration"
	"github.com/IMQS/log"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// DBConnectFlags are flags passed to OpenDB.
type DBConnectFlags int

const (
	// DBConnectFlagWipeDB causes the entire DB to erased, and re-initialized from scratch (useful for unit tests).
	DBConnectFlagWipeDB DBConnectFlags = 1 << iota
)

// Model is our base class for a GORM model.
// The default GORM Model uses int, but we prefer int64
type Model struct {
	ID        *int64     `json:"id" gorm:"primary_key"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

// DBConfig is the standard database config that we expect to find on our JSON config file.
type DBConfig struct {
	Driver   string
	Host     string
	Database string
	Username string
	Password string
}

// DSN returns a database connection string (built for Postgres only).
func (db *DBConfig) DSN() string {
	escape := func(s string) string {
		e := strings.Builder{}
		for _, r := range s {
			if r == ' ' || r == '\'' {
				e.WriteRune('\\')
			}
			e.WriteRune(r)
		}
		return e.String()
	}
	return fmt.Sprintf("host=%v user=%v password=%v dbname=%v sslmode=disable", escape(db.Host), escape(db.Username), escape(db.Password), escape(db.Database))
}

// MakeMigrations turns a sequence of SQL expression into burntsushi migrations.
// If log is not null, then the run of every migration will be logged.
func MakeMigrations(log *log.Logger, sql []string) []migration.Migrator {
	migs := []migration.Migrator{}
	for idx, str := range sql {
		version := idx
		query := str
		mig := func(tx migration.LimitedTx) error {
			if log != nil {
				summary := strings.TrimSpace(query)
				log.Infof("Running migration %v/%v: '%v...'", version+1, len(sql), summary[:40])
			}
			_, err := tx.Exec(query)
			return err
		}
		migs = append(migs, mig)
	}
	return migs
}

// OpenDB creates a new DB, or opens an existing one, and runs all the migrations before returning.
func OpenDB(log *log.Logger, driver, dsn string, migrations []migration.Migrator, flags DBConnectFlags) (*gorm.DB, error) {
	if flags&DBConnectFlagWipeDB != 0 {
		if err := DropAllTables(log, driver, dsn); err != nil {
			return nil, err
		}
	}

	// This is the common fast path, where the database has been created
	db, err := migration.Open(driver, dsn, migrations)
	if err == nil {
		db.Close()
		return gormOpen(driver, dsn)
	}

	// Automatically create the database if it doesn't already exist

	if !isDatabaseNotExist(err) {
		return nil, err
	}

	dbname, err := extractDBNameFromDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("While trying to create database, %v", err)
	}

	log.Infof("Attempting to create database %v", dbname)

	// connect to the 'postgres' database in order to create the new DB
	dsnCreate := strings.Replace(dsn, "dbname="+dbname, "dbname=postgres", -1)
	if err := createDB(driver, dsnCreate, dbname); err != nil {
		return nil, fmt.Errorf("While trying to create database '%v': %v", dbname, err)
	}
	// once again, run migrations (now that the DB has been created)
	db, err = migration.Open(driver, dsn, migrations)
	if err != nil {
		return nil, err
	}
	db.Close()
	// finally, open with gorm
	return gormOpen(driver, dsn)
}

// DropAllTables delete all tables in the given database.
// If the database does not exist, returns nil.
// This function is intended to be used by unit tests.
func DropAllTables(log *log.Logger, driver, dsn string) error {
	db, err := sql.Open(driver, dsn)
	if err == nil {
		// Force delay-connect drivers to attempt a connect now
		err = db.Ping()
	}
	if isDatabaseNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer db.Close()
	if log != nil {
		dbname, _ := extractDBNameFromDSN(dsn)
		log.Warnf("Erasing entire DB '%v'", dbname)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`
	SELECT table_name, table_schema
	FROM information_schema.tables
	WHERE
	table_schema <> 'pg_catalog' AND
	table_schema <> 'information_schema'`)
	if err != nil {
		return err
	}
	tables := []string{}
	for rows.Next() {
		var table, schema string
		if err := rows.Scan(&table, &schema); err != nil {
			return err
		}
		tables = append(tables, fmt.Sprintf(`"%v"."%v"`, schema, table))
	}
	for _, table := range tables {
		// Skip PostGIS views
		if table == `"public"."geography_columns"` ||
			table == `"public"."geometry_columns"` ||
			table == `"public"."spatial_ref_sys"` ||
			table == `"public"."raster_columns"` ||
			table == `"public"."raster_overviews"` {
			continue
		}
		//if log != nil {
		//	log.Warnf("Dropping table %v", table)
		//}
		if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %v", table)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SQLCleanIDList turns a string such as "10,34" into the string "(10,32)", so that it can be used inside an IN clause.
func SQLCleanIDList(raw string) string {
	if len(raw) == 0 {
		return "()"
	}
	res := strings.Builder{}
	res.WriteRune('(')
	parts := strings.Split(raw, ",")
	for i, t := range parts {
		id, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			continue
		}
		res.WriteString(strconv.FormatInt(id, 10))
		if i != len(parts)-1 {
			res.WriteRune(',')
		}
	}
	res.WriteRune(')')
	return res.String()
}

func gormOpen(driver, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	// Disable pluralization of tables.
	// This is just another thing to worry about when writing our own migrations, so rather disable it.
	db.SingularTable(true)
	return db, nil
}

func extractDBNameFromDSN(dsn string) (string, error) {
	matches := regexp.MustCompile("dbname=([^ ]+)").FindAllStringSubmatch(dsn, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("Failed to extract dbname= out of DSN")
	}
	return matches[0][1], nil
}

func isDatabaseNotExist(err error) bool {
	if err == nil {
		return false
	}
	return strings.Index(err.Error(), "does not exist") != -1
}

// Create a database called dbCreateName, by connecting to dsn.
func createDB(driver, dsn, dbCreateName string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("CREATE DATABASE " + dbCreateName); err != nil {
		return err
	}
	return nil
}
