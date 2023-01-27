package muskoka


type Image struct {
	ID        	int64     	`json:"id"`
	Filename  	string    	`json:"filename"`
	Size		int64		`json:"size"`
	ImageType 	ImageType 	`json:"imageType"`
}

func InitImage() {
	createImageTable()
	createImageIndices()
}

func createImageTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS images (
			id BIGSERIAL PRIMARY KEY,
			filename text NOT NULL,
			size integer NOT NULL,
            image_type_id integer references image_types NOT NULL,
			door_sample_id integer references door_samples ON DELETE CASCADE,
			gallery_sample_id integer references gallery_samples ON DELETE CASCADE,
			dealer_id integer references dealers ON DELETE CASCADE
		);`)
	if err != nil {
		panic(err)
	}
}

func createImageIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS images__filename__key ON images (lower(filename));`)
	if err != nil {
		panic(err)
	}
}
