package muskoka

import (
	"database/sql"
	"fmt"
	"github.com/kataras/iris"
	"github.com/lib/pq"
	"strings"
	"sync"
	"time"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1930Haig"
	dbname   = "muskoka"
)

var dbConn *sql.DB
var once sync.Once

func GetDBConnection() *sql.DB {
	once.Do(func() {

		connectionString := fmt.Sprintf("user=%s "+
			"password=%s dbname=%s sslmode=disable",
			user, password, dbname)

        var err error
        dbConn, err = sql.Open("postgres", connectionString)
		if err != nil {
			panic(err)
		}
	
		err = dbConn.Ping()
		if err != nil {
			panic(err)
		}

        dbConn.SetMaxOpenConns(20) // Sane default
        dbConn.SetMaxIdleConns(0)
        dbConn.SetConnMaxLifetime(time.Nanosecond)
    })
    return dbConn
}


func CloseDb() {
	if dbConn != nil {
		dbConn.Close()
	}
}

func GetColumnNameElementsFromConstraint(constraintName string) []string {
	arr := strings.Split(constraintName, "__")
	return strings.Split(arr[1], "_")
}

func GetTitleFromConstraint(constraintName string) string {
	nameElements := GetColumnNameElementsFromConstraint(constraintName)
	title := strings.Join(nameElements, " ")
	return strings.Title(title)
}

func GetCamelCaseNameFromConstraint(constraintName string) string {
	nameElements := GetColumnNameElementsFromConstraint(constraintName)

	name := ""
	for index, element := range nameElements {
		if index > 0 {
			element = strings.Title(element)
		}
		name += element
	}

	return name
}

func HandleDBError(err error) (int, map[string]interface{}) {

	switch v := err.(type) {
	default:
		return handleSQLError(v)
	case *pq.Error:
		return handlePQError(v)
	}
}

func handleSQLError(err error) (int, map[string]interface{}) {
	var statusCode int
	var errorObj map[string]interface{}

	switch err {
	case sql.ErrNoRows:
		statusCode = iris.StatusNotFound
		errorObj = map[string]interface{}{
			"error": err.Error(),
		}
		break
	default:
		statusCode = iris.StatusInternalServerError
		errorObj = map[string]interface{}{
			"error": "Unknown Error",
		}
	}

	return statusCode, errorObj
}

func handlePQError(err *pq.Error) (int, map[string]interface{}) {

	var statusCode int
	var errorObj map[string]interface{}

	if err.Code.Name() == "unique_violation" {
		statusCode = iris.StatusBadRequest
		camelCaseName := GetCamelCaseNameFromConstraint(err.Constraint)
		titleName := GetTitleFromConstraint(err.Constraint)
		errorObj = map[string]interface{}{
			"validationErrors": map[string]interface{}{
				camelCaseName: fmt.Sprintf("%s must be unique.", titleName),
			},
		}
	}

	return statusCode, errorObj
}
