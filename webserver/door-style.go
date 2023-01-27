package muskoka

import (
	"strconv"
	"sync"
	"encoding/json"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type DoorStyle struct {
	ID             int64           `json:"id"`
	DoorStyleTypes []DoorStyleType `json:"doorStyleTypes"`
	Name           string          `json:"name"`
}

func InitDoorStyle() {
	createDoorStyleTable()
	createDoorStyleIndices()
	createDoorStyleViews()
}

func createDoorStyleTable() {
	_, err := GetDBConnection().Exec(`
		CREATE TABLE IF NOT EXISTS door_styles (
			id BIGSERIAL PRIMARY KEY,
			name text NOT NULL
		);`)
	if err != nil {
		panic(err)
	}
}

func createDoorStyleIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS door_styles__name__key ON door_styles (lower(name));`)
	if err != nil {
		panic(err)
	}
}

func createDoorStyleViews() {
	createDoorStylesView()
}

func createDoorStylesView() {
	_, err := GetDBConnection().Exec(`
		CREATE OR REPLACE VIEW all_door_styles AS
		SELECT row_to_json(t)
		FROM (
			SELECT door_styles.id, door_styles.name,
			(
				SELECT array_to_json(array_agg(row_to_json(d)))
				FROM (
					SELECT door_style_types.id, door_style_types.name
					FROM door_style_door_style_types
					INNER JOIN door_style_types ON door_style_door_style_types.door_style_type_id = door_style_types.id
					WHERE door_style_door_style_types.door_style_id = door_styles.id
					ORDER BY door_style_types.name ASC
					) as d
			) as doorStyleTypes
			FROM door_styles
			ORDER BY name
		) t
	`)
	if err != nil {
		panic(err)
	}
}



func CreateDoorStyleAPI(party router.Party) {

	party.Get("/findOne/:id", findOneDoorStyleHandler)
	party.Get("", findDoorStylesHandler)
	party.Post("", insertDoorStyleHandler)
	party.Put("", updateOneDoorStyleHandler)
	party.Delete("/:id", removeOneDoorStyleHandler)
}

func findOneDoorStyleHandler(ctx context.Context) {
	id, err := ctx.URLParamInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style id"})
		return
	}

	doorStyleChannel := make(chan *DoorStyle)
	doorStyleTypesChannel := make(chan *[]DoorStyleType)
	errChannel := make(chan error, 1)

	go func() {
		doorStyle, err := FindDoorStyleFromID(id)
		if err != nil {
			errChannel <- err
		}
		doorStyleChannel <- doorStyle
	}()
	go func() {
		doorStyleTypes, err := FindDoorStyleTypesFromDoorStyleID(id)
		if err != nil {
			errChannel <- err
		}
		doorStyleTypesChannel <- doorStyleTypes
	}()

	func(doorStyle *DoorStyle, doorStyleTypes *[]DoorStyleType) {

		if err := <-errChannel; err != nil {
			statusCode, errObj := HandleDBError(err)
			ctx.StatusCode(statusCode)
			ctx.JSON(errObj)
			return
		}

		doorStyle.DoorStyleTypes = *doorStyleTypes
		ctx.StatusCode(iris.StatusOK)
		ctx.JSON(doorStyle)

	}(<-doorStyleChannel, <-doorStyleTypesChannel)
}

func FindDoorStyleFromID(id int64) (*DoorStyle, error) {
	doorStyle := DoorStyle{ID: id}
	err := GetDBConnection().QueryRow(`
		SELECT name
		FROM door_styles
		WHERE door_style_id = $1
		`, doorStyle.ID).Scan(&doorStyle.Name)
	return &doorStyle, err
}

func FindDoorStyleTypesFromDoorStyleID(id int64) (*[]DoorStyleType, error) {
	rows, err := GetDBConnection().Query(`
		SELECT door_style_types.id, door_style_types.name
		FROM door_style_door_style_types
		WHERE door_style_door_style_types.door_style_id = $1
		INNER JOIN door_style_types ON door_style_door_style_types.door_style_type_id = door_style_types.id`,
		id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	doorStyleTypes := []DoorStyleType{}
	for rows.Next() {
		doorStyleType := DoorStyleType{}
		err = rows.Scan(&doorStyleType.ID, &doorStyleType.Name)
		if err != nil {
			return nil, err
		}
		doorStyleTypes = append(doorStyleTypes, doorStyleType)
	}

	return &doorStyleTypes, nil
}

func findDoorStylesHandler(ctx context.Context) {
	doorStyles, err := FindDoorStyles()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(doorStyles)
}

func FindDoorStyles() (*[]DoorStyle, error) {

	rows, err := GetDBConnection().Query(`
		SELECT * FROM all_door_styles
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	doorStyles := []DoorStyle{}
	for rows.Next() {

		var rawMessage json.RawMessage
		err = rows.Scan(&rawMessage)
		if err != nil {
			return nil, err
		}

		Source := (*json.RawMessage)(&rawMessage)
		var doorStyle DoorStyle
		err := json.Unmarshal(*Source, &doorStyle)
		if err != nil {
			return nil, err
		}
		doorStyles = append(doorStyles, doorStyle)

		err = rows.Err()
		if err != nil {
			return nil, err
		}
		
	}

	err = rows.Close()
	if err != nil {
		return nil, err
	}

	return &doorStyles, err
}

func insertDoorStyleHandler(ctx context.Context) {
	doorStyle := &DoorStyle{}
	if err := ctx.ReadJSON(doorStyle); err != nil {
		panic(err)
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style"})
		return
	}

	// Insert Door Style
	err := GetDBConnection().QueryRow(`
		INSERT INTO door_styles (name)
		VALUES($1) returning id;`,
		doorStyle.Name).Scan(&doorStyle.ID)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	// Insert DoorStyleDoorStyleTypes
	var wg sync.WaitGroup
	waitGroupLength := len(doorStyle.DoorStyleTypes)
	errChannel := make(chan error, 1)

	wg.Add(waitGroupLength)
	finished := make(chan bool, 1)

	for i := 0; i < waitGroupLength; i++ {

		go func(i int) {
			var doorStyleDoorStyleTypeID int64
			err := GetDBConnection().QueryRow(`
				INSERT INTO door_style_door_style_types (door_style_id, door_style_type_id)
				VALUES($1,$2) returning id;`,
				doorStyle.ID, doorStyle.DoorStyleTypes[i].ID).Scan(&doorStyleDoorStyleTypeID)
			if err != nil {
				errChannel <- err
			}

			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		ctx.StatusCode(iris.StatusOK)
		ctx.JSON(doorStyle)
		break
	case err := <-errChannel:
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		break
	}
}

func updateOneDoorStyleHandler(ctx context.Context) {
	doorStyle := &DoorStyle{}
	if err := ctx.ReadJSON(doorStyle); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style"})
		return
	}

	// Update Door Style
	stmt, err := GetDBConnection().Prepare(`
		UPDATE door_styles 
		SET name=$1  
		WHERE id=$2
	`)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err := stmt.Exec(doorStyle.Name, doorStyle.ID)
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

	// Delete DoorStyleDoorStyleTypes
	stmt, err = GetDBConnection().Prepare("delete from door_style_door_style_types where door_style_id=$1")
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, err = stmt.Exec(doorStyle.ID)
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

	// Insert DoorStyleDoorStyleTypes
	var wg sync.WaitGroup
	waitGroupLength := len(doorStyle.DoorStyleTypes)
	errChannel := make(chan error, 1)

	wg.Add(waitGroupLength)
	finished := make(chan bool, 1)

	for i := 0; i < waitGroupLength; i++ {

		go func(i int) {
			var doorStyleDoorStyleTypeID int64
			err := GetDBConnection().QueryRow(`
				INSERT INTO door_style_door_style_types (door_style_id, door_style_type_id)
				VALUES($1,$2) returning id;`,
				doorStyle.ID, doorStyle.DoorStyleTypes[i].ID).Scan(&doorStyleDoorStyleTypeID)
			if err != nil {
				errChannel <- err
			}

			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		ctx.StatusCode(iris.StatusOK)
		ctx.JSON(map[string]interface{}{})
		break
	case err := <-errChannel:
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		break
	}
}

func removeOneDoorStyleHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read door style id"})
		return
	}

	// Delete Door Style
	stmt, err := GetDBConnection().Prepare("delete from door_styles where id=$1")
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
		ctx.JSON(map[string]interface{}{"error": "No door style found"})
		return
	}

	// Delete DoorStyleDoorStyleTypes
	stmt, err = GetDBConnection().Prepare("delete from door_style_door_style_types where door_style_id=$1")
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

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}
