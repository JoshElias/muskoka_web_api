package muskoka

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
	"strconv"
)

type Wood struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func InitWood() {
	createWoodTable()
	createWoodIndices()
}

func createWoodTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS wood (
		id BIGSERIAL PRIMARY KEY,
		name text NOT NULL
	);`)
	if err != nil {
		panic(err)
	}
}

func createWoodIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS wood__name__key ON wood (lower(name));`)
	if err != nil {
		panic(err)
	}
}

func CreateWoodAPI(party router.Party) {
	party.Get("/findOne/:id", findOneWoodHandler)
	party.Get("", findWoodHandler)
	party.Post("", insertWoodHandler)
	party.Put("", updateOneWoodHandler)
	party.Delete("/:id", removeOneWoodHandler)
}

func findOneWoodHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read wood id"})
		return
	}

	wood := Wood{ID: id}
	err = GetDBConnection().QueryRow(`
		SELECT name 
		FROM wood
		WHERE id = $1`,
		id).Scan(&wood.Name)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(wood)
}

func findWoodHandler(ctx context.Context) {
	rows, err := GetDBConnection().Query(`
		SELECT id, name 
		FROM wood
		ORDER BY name`)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}
	defer rows.Close()

	woods := []Wood{}
	for rows.Next() {
		wood := Wood{}
		err = rows.Scan(&wood.ID, &wood.Name)
		if err != nil {
			statusCode, errObj := HandleDBError(err)
			ctx.StatusCode(statusCode)
			ctx.JSON(errObj)
			return
		}
		woods = append(woods, wood)
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(woods)
}

func insertWoodHandler(ctx context.Context) {
	wood := &Wood{}
	if err := ctx.ReadJSON(wood); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read wood"})
		return
	}

	err := GetDBConnection().QueryRow(`INSERT INTO wood (name)
		VALUES($1) returning id;`, wood.Name).Scan(&wood.ID)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(wood)
}

func removeOneWoodHandler(ctx context.Context) {

	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read wood id"})
		return
	}

	stmt, err := GetDBConnection().Prepare("delete from wood where id=$1")
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
		ctx.JSON(map[string]interface{}{"error": "No colour found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}

func updateOneWoodHandler(ctx context.Context) {
	wood := &Wood{}
	if err := ctx.ReadJSON(wood); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read wood"})
		return
	}

	stmt, err := GetDBConnection().Prepare(`
		UPDATE wood 
		SET name=$1  
		WHERE id=$2
	`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(wood.Name, wood.ID)
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
		ctx.JSON(map[string]interface{}{"error": "No wood found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{})
}
