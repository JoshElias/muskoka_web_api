package muskoka

import (
	"net/url"
	"strconv"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type Dealer struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Link        string `json:"link"`
	Location    string `json:"location"`
	PhoneNumber int64  `json:"phoneNumber"`
	Email       string `json:"email"`
	OrderNum    int64  `json:"orderNum"`
	Image       Image  `json:"image"`
}

func InitDealer() {
	createDealerTable()
	createDealerIndices()
}

func createDealerTable() {
	_, err := GetDBConnection().Exec(`CREATE TABLE IF NOT EXISTS dealers (
			id BIGSERIAL PRIMARY KEY,
			name text NOT NULL,
			link text,
			location text,
			phone_num BIGINT,
			email string,
			order_num integer NOT NULL
		);`)
	if err != nil {
		panic(err)
	}
}

func createDealerIndices() {
	_, err := GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS dealers__name__key ON dealers (lower(name));`)
	if err != nil {
		panic(err)
	}

	//_, err = GetDBConnection().Exec(`CREATE UNIQUE INDEX IF NOT EXISTS dealers__order_num__key ON dealers (order_num) DEFERRABLE;`)
	//if err != nil {
	//panic(err)
	//}
}

func CreateDealerAPI(party router.Party) {
	party.Get("/findOne/:id", findOneDealerHandler)
	party.Get("", findDealersHandler)
	party.Post("", insertDealerHandler)
	party.Put("", updateOneDealerHandler)
	party.Delete("/:id", removeOneDealerHandler)
}

func findOneDealerHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read dealer id"})
		return
	}

	dealer := Dealer{ID: id}
	err = GetDBConnection().QueryRow(`
		SELECT dealers.name, dealers.link, dealers.location, dealers.phone_num, 
				dealers.email, dealers.order_num,
			images.id, images.filename, images.size
		FROM dealers
		INNER JOIN images ON dealers.id = images.dealer_id
		WHERE dealers.id = $1
		`, dealer.ID).Scan(
		&dealer.Name, &dealer.Link, &dealer.Location, &dealer.PhoneNumber,
		&dealer.Email, &dealer.OrderNum,
		&dealer.Image.ID, &dealer.Image.Filename, &dealer.Image.Size)
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(dealer)
}

func findDealersHandler(ctx context.Context) {
	dealers, err := FindDealers()
	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(dealers)
}

func FindDealers() (*[]Dealer, error) {
	rows, err := GetDBConnection().Query(`
		SELECT dealers.id, dealers.name, dealers.link, dealers.location, dealers.phone_num,
				dealers.email, dealers.order_num,
			images.id, images.filename, images.size
		FROM dealers
		INNER JOIN images ON dealers.id = images.dealer_id
		ORDER BY dealers.order_num ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dealers := []Dealer{}
	for rows.Next() {
		dealer := Dealer{}
		err = rows.Scan(
			&dealer.ID, &dealer.Name, &dealer.Link, &dealer.Location, &dealer.PhoneNumber,
			&dealer.Email, &dealer.OrderNum,
			&dealer.Image.ID, &dealer.Image.Filename, &dealer.Image.Size)
		if err != nil {
			return nil, err
		}
		dealers = append(dealers, dealer)
	}

	return &dealers, nil
}

func insertDealerHandler(ctx context.Context) {
	dealer := &Dealer{}
	if err := ctx.ReadJSON(dealer); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read dealer"})
		return
	}

	dealer.Image.Filename = url.QueryEscape(dealer.Image.Filename)

	// Create Dealer
	err := GetDBConnection().QueryRow(`
		INSERT INTO dealers (name, link, location, phone_num, email, order_num)
		VALUES($1,$2,$3,$4,$5,$6) returning id;`,
		dealer.Name, dealer.Link, dealer.Location, dealer.PhoneNumber, dealer.Email, dealer.OrderNum).Scan(&dealer.ID)

	if err != nil {
		statusCode, errObj := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	// Create new image
	err = GetDBConnection().QueryRow(`
		INSERT INTO images (filename, size, image_type_id, dealer_id)
		VALUES($1,$2,$3,$4) 
		returning id;`,
		dealer.Image.Filename, dealer.Image.Size, dealer.Image.ImageType.ID, dealer.ID).Scan(&dealer.Image.ID)
	if err != nil {
		statusCode, result := HandleDBError(err)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(dealer)
}

func removeOneDealerHandler(ctx context.Context) {
	id, err := ctx.Params().GetInt64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read dealer id"})
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
		WHERE dealer_id=$1`,
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
		ctx.JSON(map[string]interface{}{"error": "No images found for dealer"})
		return
	}

	// Delete doorsample from database
	stmt, err = tx.Prepare(`delete from dealers where id=$1`)
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
		ctx.JSON(map[string]interface{}{"error": "No dealers found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	idString := strconv.FormatInt(id, 10)
	ctx.JSON(map[string]interface{}{
		"id": idString,
	})
}

func updateOneDealerHandler(ctx context.Context) {
	dealer := &Dealer{}
	if err := ctx.ReadJSON(dealer); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read dealer"})
		return
	}

	// Get the old order number and image info to see if they've been changed
	txn, dbErr := GetDBConnection().Begin()
	if dbErr != nil {
		statusCode, result := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	var oldOrderNum int64
	dbErr = txn.QueryRow(`
		SELECT order_num
		FROM dealers
		WHERE id = $1`,
		dealer.ID).Scan(&oldOrderNum)
	if dbErr != nil {
		statusCode, errObj := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
	}

	oldImage := &Image{}
	dbErr = txn.QueryRow(`
		SELECT filename, size 
		FROM images
		WHERE dealer_id = $1`,
		dealer.ID).Scan(&oldImage.Filename, &oldImage.Size)
	if dbErr != nil {
		statusCode, errObj := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
		return
	}

	dbErr = txn.Commit()
	if dbErr != nil {
		statusCode, errObj := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(errObj)
	}

	// Did we update the orderNum? Swap those first because of the unique constraint
	if dealer.OrderNum != oldOrderNum {

		// Get the id of the old order number owner
		// stupid I know
		var otherDealerID int64
		dbErr = GetDBConnection().QueryRow(`
			SELECT id
			FROM dealers
			WHERE order_num = $1`,
			dealer.OrderNum).Scan(&otherDealerID)
		if dbErr != nil {
			statusCode, errObj := HandleDBError(dbErr)
			ctx.StatusCode(statusCode)
			ctx.JSON(errObj)
			return
		}

		// Swap orderNums with however owned that order num
		stmt, dbErr := GetDBConnection().Prepare(`
			UPDATE dealers dst
			SET order_num = src.order_num
			FROM dealers src
			WHERE dst.id IN($1,$2)
			AND src.id IN($1,$2)
			AND dst.id <> src.id;
		`)
		if dbErr != nil {
			statusCode, result := HandleDBError(dbErr)
			ctx.StatusCode(statusCode)
			ctx.JSON(result)
			return
		}

		res, dbErr := stmt.Exec(otherDealerID, dealer.ID)
		if dbErr != nil {
			statusCode, result := HandleDBError(dbErr)
			ctx.StatusCode(statusCode)
			ctx.JSON(result)
			return
		}

		dbErr = stmt.Close()
		if dbErr != nil {
			statusCode, errObj := HandleDBError(dbErr)
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
			ctx.JSON(map[string]interface{}{"error": "No dealer found with old order number. Update cannot create new order numbers"})
			return
		}
	}

	// Did we update the image?
	if dealer.Image.Filename != oldImage.Filename || dealer.Image.Size != oldImage.Size {

		// Update the database
		stmt, dbErr := GetDBConnection().Prepare(`
			UPDATE images 
			SET filename = $1, size = $2
			WHERE id = $3
		`)
		if dbErr != nil {
			statusCode, result := HandleDBError(dbErr)
			ctx.StatusCode(statusCode)
			ctx.JSON(result)
			return
		}

		res, err := stmt.Exec(dealer.Image.Filename, dealer.Image.Size, dealer.Image.ID)
		if dbErr != nil {
			statusCode, result := HandleDBError(dbErr)
			ctx.StatusCode(statusCode)
			ctx.JSON(result)
			return
		}

		dbErr = stmt.Close()
		if dbErr != nil {
			statusCode, errObj := HandleDBError(dbErr)
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

	// Update Dealer in DB
	// minus the order_num
	// already did that...
	stmt, dbErr := GetDBConnection().Prepare(`
		UPDATE dealers 
		SET name=$1, link=$2, location=$3, phone_num=$4, email=$5
		WHERE id=$6
	`)
	if dbErr != nil {
		statusCode, result := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	res, dbErr := stmt.Exec(dealer.Name, dealer.Link, dealer.Location,
		dealer.PhoneNumber, dealer.Email, dealer.ID)
	if dbErr != nil {
		statusCode, result := HandleDBError(dbErr)
		ctx.StatusCode(statusCode)
		ctx.JSON(result)
		return
	}

	dbErr = stmt.Close()
	if dbErr != nil {
		statusCode, errObj := HandleDBError(dbErr)
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
		ctx.JSON(map[string]interface{}{"error": "No dealer found"})
		return
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{})
}
