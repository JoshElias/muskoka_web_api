package muskoka

import (
	"strconv"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type ImageType struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	IsSpecificDimension bool   `json:"isSpecificDimension"`
	Width               int    `json:"width"`
	Height              int    `json:"height"`
}

func InitImageType() {
	createImageTypeTable()
	createImageTypeIndices()
}

func createImageTypeTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS image_types (
			id BIGSERIAL PRIMARY KEY,
			name text NOT NULL,
			is_specific_dimension BOOLEAN DEFAULT FALSE,
            width smallint DEFAULT 0,
            height smallint DEFAULT 0
		);`)
	if err != nil {
		panic(err)
	}
}

func createImageTypeIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS image_types__name__key ON image_types (lower(name));`)
	if err != nil {
		panic(err)
	}
}

func CreateImageTypeAPI(party router.Party) {
	party.Get("findOne/:id", findOneImageTypeHandler)
	party.Get("", findImageTypesHandler)
	party.Post("", insertImageTypeHandler)
	party.Put("", updateOneImageTypeHandler)
	party.Delete("/:id", removeOneImageTypeHandler)
}

func findOneImageTypeHandler(ctx context.Context) {
	id, err := ctx.URLParamInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door sample"})
		return
	}

	imageType := ImageType{ID: id}
	err = GetDBConnection().QueryRow(`
		SELECT name, is_specific_dimension, width, height
		FROM image_types
		WHERE id = $1
		ORDER BY name ASC
		`, imageType.ID).Scan(
		&imageType.Name, &imageType.IsSpecificDimension, &imageType.Width, &imageType.Height)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(imageType)
}

func findImageTypesHandler(ctx context.Context) {
	rows, err := GetDBConnection().Query("SELECT id, name, is_specific_dimension, width, height FROM image_types")
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}
	defer rows.Close()

	imageTypes := []ImageType{}
	for rows.Next() {
		imageType := ImageType{}
		err = rows.Scan(&imageType.ID, &imageType.Name,
			&imageType.IsSpecificDimension, &imageType.Width, &imageType.Height)
		if err != nil {
			statusCode, errObj := HandleDBError(err)
			ctx.StatusCode(statusCode)
			ctx.JSON(errObj)
			return
		}
		imageTypes = append(imageTypes, imageType)
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(imageTypes)
}

func insertImageTypeHandler(ctx context.Context) {

	imageType := &ImageType{}
	if err := ctx.ReadJSON(imageType); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read image type"})
		return
	}

	err := GetDBConnection().QueryRow(`
		INSERT INTO image_types (name, is_specific_dimension, width, height)
		VALUES($1,$2,$3,$4) 
		returning id;`,
		imageType.Name, imageType.IsSpecificDimension, imageType.Width, imageType.Height).Scan(&imageType.ID)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(imageType)
}

func updateOneImageTypeHandler(ctx context.Context) {
	imageType := &ImageType{}
	if err := ctx.ReadJSON(imageType); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read image type"})
		return
	}

	stmt, err := GetDBConnection().Prepare(`
		UPDATE image_types 
		SET name=$1, is_specific_dimension=$2, width=$3, height=$4  
		WHERE id=$5
	`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(imageType.Name, imageType.IsSpecificDimension,
		imageType.Width, imageType.Height, imageType.ID)
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

func removeOneImageTypeHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read image type id"})
		return
	}

	stmt, err := GetDBConnection().Prepare("delete from image_types where id=$1")
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
		ctx.JSON(map[string]interface{}{"error": "No image type found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}
