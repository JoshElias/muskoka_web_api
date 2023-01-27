package muskoka

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type DoorSample struct {
	ID        int64     `json:"id"`
	DoorStyle DoorStyle `json:"doorStyle"`
	Wood      Wood      `json:"wood"`
	Colour    Colour    `json:"colour"`
	Image     Image     `json:"image"`
}

func InitDoorSample() {
	createDoorSampleTable()
}

func createDoorSampleTable() {
	_, err := GetDBConnection().Exec(`
		CREATE TABLE IF NOT EXISTS door_samples (
			id BIGSERIAL PRIMARY KEY,
			door_style_id integer references door_styles NOT NULL,
			wood_id integer references wood NOT NULL,
			colour_id integer references colours NOT NULL
		);`)
	if err != nil {
		panic(err)
	}
}

func CreateDoorSampleAPI(party router.Party) {

	party.Get("/findOne/:id", findOneDoorSampleHandler)
	party.Get("", findDoorSamplesHandler)
	party.Post("", insertDoorSampleHandler)
	party.Put("", updateOneDoorSampleHandler)
	party.Delete("/:id", removeOneDoorSampleHandler)
}

func findOneDoorSampleHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door sample id"})
		return
	}

	doorSample := DoorSample{ID: id}
	err = GetDBConnection().QueryRow(`
		SELECT door_styles.id, door_styles.name, wood.id, wood.name, colours.id, colours.name, 
			images.id, images.filename, images.size
		FROM door_samples
		INNER JOIN door_styles ON door_samples.door_style_id = door_styles.id
		INNER JOIN wood ON door_samples.wood_id = wood.id
		INNER JOIN colours ON door_samples.colour_id = colours.id
		INNER JOIN images ON door_samples.id = images.door_sample_id
		WHERE door_samples.id = $1
		`, doorSample.ID).Scan(
		&doorSample.DoorStyle.ID, &doorSample.DoorStyle.Name, &doorSample.Wood.ID, &doorSample.Wood.Name,
		&doorSample.Colour.ID, &doorSample.Colour.Name, &doorSample.Image.ID, &doorSample.Image.Filename,
		&doorSample.Image.Size)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorSample)
}

type DoorSampleSearch struct {
	ColourIDs    []int  `json:"colourIds"`
	WoodIDs      []int  `json:"woodIds"`
	DoorStyleIDs []int  `json:"doorStyleIds"`
	SearchText   string `json:"searchText"`
}

func findDoorSamplesHandler(ctx context.Context) {

	coloursIDsString := ctx.URLParam("colourIds")
	woodIDsString := ctx.URLParam("woodIds")
	doorStyleIDsString := ctx.URLParam("doorStyleIds")
	searchText := ctx.URLParam("searchText")

	doorSampleSearch := DoorSampleSearch{}
	json.Unmarshal([]byte(coloursIDsString), &doorSampleSearch.ColourIDs)
	json.Unmarshal([]byte(woodIDsString), &doorSampleSearch.WoodIDs)
	json.Unmarshal([]byte(doorStyleIDsString), &doorSampleSearch.DoorStyleIDs)
	doorSampleSearch.SearchText = searchText

	doorSamples, err := FindDoorSamples(&doorSampleSearch)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorSamples)
}

func FindDoorSamples(search *DoorSampleSearch) (*[]DoorSample, error) {

	numOfArgs := len(search.ColourIDs) + len(search.WoodIDs) + len(search.DoorStyleIDs)
	if len(search.SearchText) > 0 {
		search.SearchText = strings.ToLower(search.SearchText)
		numOfArgs++
	}

	argumentCounter := 1
	whereArguments := make([]interface{}, 0)
	whereQueries := []string{}
	for _, element := range search.ColourIDs {
		idQuery := fmt.Sprintf("door_samples.colour_id = $%[1]d", argumentCounter)
		whereQueries = append(whereQueries, idQuery)
		whereArguments = append(whereArguments, element)
		argumentCounter++
	}

	for _, element := range search.WoodIDs {
		idQuery := fmt.Sprintf("door_samples.wood_id = $%[1]d", argumentCounter)
		whereQueries = append(whereQueries, idQuery)
		whereArguments = append(whereArguments, element)
		argumentCounter++
	}

	for _, element := range search.DoorStyleIDs {
		idQuery := fmt.Sprintf("door_samples.door_style_id = $%[1]d", argumentCounter)
		whereQueries = append(whereQueries, idQuery)
		whereArguments = append(whereArguments, element)
		argumentCounter++
	}

	if len(search.SearchText) > 0 {
		searchQuery := fmt.Sprintf("LOWER(colours.name) LIKE '%%' || $%[1]d || '%%'", argumentCounter)
		whereQueries = append(whereQueries, searchQuery)
		searchQuery = fmt.Sprintf("LOWER(wood.name) LIKE '%%' || $%[1]d || '%%'", argumentCounter)
		whereQueries = append(whereQueries, searchQuery)
		searchQuery = fmt.Sprintf("LOWER(door_styles.name) LIKE '%%' || $%[1]d || '%%'", argumentCounter)
		whereQueries = append(whereQueries, searchQuery)

		whereArguments = append(whereArguments, search.SearchText)
		argumentCounter++
	}

	whereQuery := ""
	if len(whereQueries) > 0 {
		whereQuery = "WHERE " + strings.Join(whereQueries, " OR ") + " "
	}

	doorSampleQuery := fmt.Sprintf(`
		SELECT door_samples.id,
			door_styles.id, door_styles.name, 
			wood.id, wood.name, 
			colours.id, colours.name, 
			images.id, images.filename, images.size
		FROM door_samples
		INNER JOIN door_styles ON door_samples.door_style_id = door_styles.id
		INNER JOIN wood ON door_samples.wood_id = wood.id
		INNER JOIN colours ON door_samples.colour_id = colours.id
		INNER JOIN images ON door_samples.id = images.door_sample_id
		%[1]s
		ORDER BY images.filename ASC`, whereQuery)
	var rows *sql.Rows
	var err error
	if len(whereArguments) > 0 {
		rows, err = GetDBConnection().Query(doorSampleQuery, whereArguments...)
	} else {
		rows, err = GetDBConnection().Query(doorSampleQuery)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	doorSamples := []DoorSample{}
	for rows.Next() {
		doorSample := DoorSample{}
		err = rows.Scan(&doorSample.ID,
			&doorSample.DoorStyle.ID, &doorSample.DoorStyle.Name,
			&doorSample.Wood.ID, &doorSample.Wood.Name,
			&doorSample.Colour.ID, &doorSample.Colour.Name,
			&doorSample.Image.ID, &doorSample.Image.Filename, &doorSample.Image.Size)
		if err != nil {
			return nil, err
		}
		doorSamples = append(doorSamples, doorSample)
	}

	return &doorSamples, nil
}

func insertDoorSampleHandler(ctx context.Context) {
	doorSample := &DoorSample{}
	if err := ctx.ReadJSON(doorSample); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door sample"})
		return
	}

	doorSample.Image.Filename = url.QueryEscape(doorSample.Image.Filename)

	// Create Door Sample
	var err error
	if doorSample.Colour.ID < 1 {
		err = GetDBConnection().QueryRow(`
			INSERT INTO door_samples (door_style_id, wood_id, colour_id)
			VALUES($1,$2,$3) returning id;`,
			doorSample.DoorStyle.ID, doorSample.Wood.ID, nil).Scan(&doorSample.ID)
	} else {
		err = GetDBConnection().QueryRow(`
			INSERT INTO door_samples (door_style_id, wood_id, colour_id)
			VALUES($1,$2,$3) returning id;`,
			doorSample.DoorStyle.ID, doorSample.Wood.ID, doorSample.Colour.ID).Scan(&doorSample.ID)
	}

	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	// Create new image first
	err = GetDBConnection().QueryRow(`
		INSERT INTO images (filename, size, image_type_id, door_sample_id)
		VALUES($1,$2,$3,$4) 
		returning id;`,
		doorSample.Image.Filename, doorSample.Image.Size, doorSample.Image.ImageType.ID, doorSample.ID).Scan(&doorSample.Image.ID)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorSample)
}

func removeOneDoorSampleHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door sample id"})
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
		WHERE door_sample_id=$1`,
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
		ctx.JSON(map[string]interface{}{"error": "No images found for doorsample found"})
		return
	}

	// Delete doorsample from database
	stmt, err = tx.Prepare(`delete from door_samples where id=$1`)
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
		ctx.JSON(map[string]interface{}{"error": "No door samples found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}

func updateOneDoorSampleHandler(ctx context.Context) {
	doorSample := &DoorSample{}
	if err := ctx.ReadJSON(doorSample); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door sample"})
		return
	}

	// Are we updating the image?
	oldImage := &Image{}
	err := GetDBConnection().QueryRow(`
		SELECT filename, size 
		FROM images
		WHERE door_sample_id = $1`,
		doorSample.ID).Scan(&oldImage.Filename, &oldImage.Size)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}


	if doorSample.Image.Filename != oldImage.Filename || doorSample.Image.Size != oldImage.Size {

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

		res, err := stmt.Exec(doorSample.Image.Filename, doorSample.Image.Size, doorSample.Image.ID)
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
			ctx.JSON(map[string]interface{}{"error": "No iamge found for door sample id"})
			return
		}

		// Update S3
		err = deleteS3Object(oldImage.Filename)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.JSON(err)
		}
	}

	// Update Door Sample
	stmt, dbErr := GetDBConnection().Prepare(`
		UPDATE door_samples 
		SET door_style_id=$1, wood_id=$2, colour_id=$3
		WHERE id=$4
	`)
	if dbErr != nil {
		statusCode, result := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, dbErr := stmt.Exec(doorSample.DoorStyle.ID, doorSample.Wood.ID,
		doorSample.Colour.ID, doorSample.ID)
	if dbErr != nil {
		statusCode, result := HandleDBError(dbErr)
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

	affect, dbErr := res.RowsAffected()
	if dbErr != nil {
		statusCode, result := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	if affect < 1 {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.JSON(map[string]interface{}{"error": "No door_sample found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{})
}
