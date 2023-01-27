package muskoka

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"strconv"
)

type DoorStyleType struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func InitDoorStyleType() {
	createDoorStyleTypeTable()
	createUniqueNameIndex()
}

func createDoorStyleTypeTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS door_style_types (
		id BIGSERIAL PRIMARY KEY,
		name text NOT NULL);`)
	if err != nil {
		panic(err)
	}
}

func createUniqueNameIndex() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS door_style_types__name__key ON door_style_types (lower(name));`)
	if err != nil {
		panic(err)
	}
}

func CreateDoorStyleTypeAPI(party router.Party) {

	party.Get("/findOne/:id", findOneDoorStyleTypeHandler)
	party.Get("", findDoorStyleTypesHandler)
	party.Post("", insertDoorStyleTypeHandler)
	party.Put("", updateOneDoorStyleTypeHandler)
	party.Delete("/:id", removeOneDoorStyleTypeHandler)
}

func findOneDoorStyleTypeHandler(ctx context.Context) {

	id, err := ctx.URLParamInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style type id"})
		return
	}

	doorStyleType := DoorStyleType{ID: id}
	err = GetDBConnection().QueryRow(`SELECT name FROM door_style_types
			WHERE id = $1`, doorStyleType.ID).Scan(&doorStyleType.Name)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorStyleType)
}

func findDoorStyleTypesHandler(ctx context.Context) {
	rows, err := GetDBConnection().Query(`
		SELECT id, name 
		FROM door_style_types
		ORDER BY name`)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}
	defer rows.Close()

	doorStyleTypes := []DoorStyleType{}
	for rows.Next() {
		doorStyleType := DoorStyleType{}
		err = rows.Scan(&doorStyleType.ID, &doorStyleType.Name)
		if err != nil {
			statusCode, errObj := HandleDBError(err)
			ctx.StatusCode(statusCode)
			ctx.JSON(errObj)
			return
		}
		doorStyleTypes = append(doorStyleTypes, doorStyleType)
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorStyleTypes)
}

func insertDoorStyleTypeHandler(ctx context.Context) {
	doorStyleType := &DoorStyleType{}
	if err := ctx.ReadJSON(doorStyleType); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style type"})
		return
	}

	err := GetDBConnection().QueryRow(`INSERT INTO door_style_types (name)
		VALUES($1) returning id;`, doorStyleType.Name).Scan(&doorStyleType.ID)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorStyleType)
}

func removeOneDoorStyleTypeHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style type id"})
		return
	}

	stmt, err := GetDBConnection().Prepare("delete from door_style_types where id=$1")
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(id)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	err = stmt.Close()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	affect, err := res.RowsAffected()
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	if affect < 1 {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.JSON(map[string]interface{}{"error": "No door style type found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}

func updateOneDoorStyleTypeHandler(ctx context.Context) {
	doorStyleType := &Colour{}
	if err := ctx.ReadJSON(doorStyleType); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style type"})
		return
	}

	stmt, err := GetDBConnection().Prepare(`
		UPDATE door_style_types 
		SET name=$1  
		WHERE id=$2
	`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(doorStyleType.Name, doorStyleType.ID)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	err = stmt.Close()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	affect, err := res.RowsAffected()
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	if affect < 1 {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.JSON(map[string]interface{}{"error": "No door style type found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{})
}
