package muskoka

type DoorStyleDoorStyleType struct {
	ID              int64 `json:"id"`
	DoorStyleID     int64 `json:"doorStyleId"`
	DoorStyleTypeID int64 `json:"doorStyleTypeId`
}

func InitDoorStyleDoorStyleType() {
	createDoorStyleDoorStyleTypesTable()
}

func createDoorStyleDoorStyleTypesTable() {
	_, err := GetDBConnection().Exec(`
		CREATE TABLE IF NOT EXISTS door_style_door_style_types (
			id BIGSERIAL PRIMARY KEY,
            door_style_id integer references door_styles ON DELETE CASCADE NOT NULL,
        	door_style_type_id integer references door_style_types ON DELETE CASCADE NOT NULL
        );`)
	if err != nil {
		panic(err)
	}
}
