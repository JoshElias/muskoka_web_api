package muskoka

import (
	"net/url"
	"strconv"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type GallerySample struct {
	ID    int64  `json:"id"`
	Image Image  `json:"image"`
}

func InitGallerySample() {
	createGallerySampleTable()
}

func createGallerySampleTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS gallery_samples (
		id BIGSERIAL PRIMARY KEY
	);`)
	if err != nil {
		panic(err)
	}
}

func CreateGallerySampleAPI(party router.Party) {

	party.Get("/findOne/:id", findOneGallerySampleHandler)
	party.Get("", findGallerySamplesHandler)
	party.Put("", updateOneGallerySampleHandler)
	party.Post("", insertGallerySampleHandler)
	party.Delete("/:id", removeOneGallerySampleHandler)
}

func findOneGallerySampleHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read gallery sample id"})
		return
	}

	gallerySample := GallerySample{ID: id}
	err = GetDBConnection().QueryRow(`
		SELECT images.id, images.filename, images.size
		FROM gallery_samples
		INNER JOIN images ON gallery_samples.id = images.gallery_sample_id
		WHERE gallery_samples.id = $1
		`, gallerySample.ID).Scan(&gallerySample.Image.ID, &gallerySample.Image.Filename,
			&gallerySample.Image.Size)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(gallerySample)
}

func findGallerySamplesHandler(ctx context.Context) {
	gallerySamples, err := FindGallerySamples()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(gallerySamples)
}

func FindGallerySamples() (*[]GallerySample, error) {
	rows, err := GetDBConnection().Query(`
		SELECT gallery_samples.id, images.id, images.filename, images.size
		FROM gallery_samples
		INNER JOIN images ON gallery_samples.id = images.gallery_sample_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gallerySamples := []GallerySample{}
	for rows.Next() {
		gallerySample := GallerySample{}
		err = rows.Scan(&gallerySample.ID, &gallerySample.Image.ID,
			&gallerySample.Image.Filename, &gallerySample.Image.Size)
		if err != nil {
			return nil, err
		}
		gallerySamples = append(gallerySamples, gallerySample)
	}

	return &gallerySamples, nil
}

func insertGallerySampleHandler(ctx context.Context) {
	gallerySample := &GallerySample{}
	if err := ctx.ReadJSON(gallerySample); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read gallery sample"})
		return
	}

	gallerySample.Image.Filename = url.QueryEscape(gallerySample.Image.Filename)

	// Create Gallery Sample
	err := GetDBConnection().QueryRow(`
		INSERT INTO gallery_samples (id) VALUES (DEFAULT) returning id;
		`).Scan(&gallerySample.ID)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	// Create new image 
	err = GetDBConnection().QueryRow(`
		INSERT INTO images (filename, size, image_type_id, gallery_sample_id)
		VALUES($1,$2,$3,$4) 
		returning id;`,
		gallerySample.Image.Filename, gallerySample.Image.Size, gallerySample.Image.ImageType.ID, 
			gallerySample.ID).Scan(&gallerySample.Image.ID)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(gallerySample)
}

func removeOneGallerySampleHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read gallery sample id"})
		return
	}

	tx, err := GetDBConnection().Begin()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	// Get image filename
	filename := ""
	dbErr := tx.QueryRow(`
		SELECT filename 
		FROM images
		WHERE gallery_sample_id=$1`,
		id).Scan(&filename)
	if dbErr != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	// Delete from S3
	err = deleteS3Object(filename)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(err)
	}

	// Delete Image from database
	stmt, err := tx.Prepare(`delete from images where filename=$1`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(filename)
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
		ctx.JSON(map[string]interface{}{"error": "No images found for gallerysample found"})
		return
	}

	// Delete doorsample from database
	stmt, err = tx.Prepare(`delete from gallery_samples where id=$1`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err = stmt.Exec(id)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	err = stmt.Close()
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	err = tx.Commit()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	affect, err = res.RowsAffected()
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	if affect < 1 {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.JSON(map[string]interface{}{"error": "No gallery samples found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}

func updateOneGallerySampleHandler(ctx context.Context) {
	gallerySample := &GallerySample{}
	if err := ctx.ReadJSON(gallerySample); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read gallery sample"})
		return
	}

	// Are we updating the image?
	oldImage := &Image{}
	err := GetDBConnection().QueryRow(`
		SELECT filename, size 
		FROM images
		WHERE gallery_sample_id = $1`,
		gallerySample.ID).Scan(&oldImage.Filename, &oldImage.Size)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	if gallerySample.Image.Filename != oldImage.Filename || gallerySample.Image.Size != oldImage.Size {

		// Update the database
		stmt, err := GetDBConnection().Prepare(`
			UPDATE images 
			SET filename = $1, size = $2
			WHERE id = $3
		`)
		if err != nil {
			statusCode, result := HandleDBError(err)
			ctx.StatusCode(statusCode)
			ctx.JSON(result)
			return
		}

		res, err := stmt.Exec(gallerySample.Image.Filename, gallerySample.Image.Size, gallerySample.Image.ID)
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
			ctx.JSON(map[string]interface{}{"error": "No iamge found for gallery sample id"})
			return
		}

		// Update S3
		err = deleteS3Object(oldImage.Filename)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.JSON(err)
		}
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{})
}
