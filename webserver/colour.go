package muskoka

import (
	"strconv"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type Colour struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func InitColour() {
	createColourTable()
	createColourIndices()
}

func createColourTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS colours (
			id BIGSERIAL PRIMARY KEY,
			name text NOT NULL
		);`)
	if err != nil {
		panic(err)
	}
}

func createColourIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS colours__name__key ON colours (lower(name));`)
	if err != nil {
		panic(err)
	}
}

func CreateColourAPI(party router.Party) {
	party.Get("/findOne/:id", findOneColourHandler)
	party.Get("", findColoursHandler)
	party.Post("", insertColourHandler)
	party.Put("", updateOneColourHandler)
	party.Delete("/:id", removeOneColourHandler)
}

func findOneColourHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read colour id"})
		return
	}

	colour := Colour{ID: id}
	dbErr := GetDBConnection().QueryRow(`
		SELECT name 
		FROM colours
		WHERE id = $1`,
		colour.ID).Scan(&colour.Name)
	if dbErr != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(colour)
}

func findColoursHandler(ctx context.Context) {
	rows, err := GetDBConnection().Query(`
		SELECT id, name 
		FROM colours
		ORDER BY name`)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}
	defer rows.Close()

	colours := []Colour{}
	for rows.Next() {
		colour := Colour{}
		err = rows.Scan(&colour.ID, &colour.Name)
		if err != nil {
			statusCode, errObj := HandleDBError(err)
			ctx.StatusCode(statusCode)
			ctx.JSON(errObj)
			return
		}
		colours = append(colours, colour)
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(colours)
}

func insertColourHandler(ctx context.Context) {
	colour := &Colour{}
	if err := ctx.ReadJSON(colour); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read colour"})
		return
	}

	err := GetDBConnection().QueryRow(`
		INSERT INTO colours (name)
		VALUES($1) returning id;`,
		colour.Name).Scan(&colour.ID)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(colour)
}

func updateOneColourHandler(ctx context.Context) {
	colour := &Colour{}
	if err := ctx.ReadJSON(colour); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read colour"})
		return
	}

	stmt, err := GetDBConnection().Prepare(`
		UPDATE colours 
		SET name=$1  
		WHERE id=$2
	`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(colour.Name, colour.ID)
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
	ctx.JSON(map[string]interface{}{})
}

func removeOneColourHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read colour id"})
		return
	}

	stmt, err := GetDBConnection().Prepare("delete from colours where id=$1")
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
