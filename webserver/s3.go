package muskoka

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type SignedURLRequest struct {
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
}

var s3Service *s3.S3

func InitS3() {
	s3Service = s3.New(session.New(&aws.Config{Region: aws.String("ca-central-1")}))
}

func CreateUploadAPI(party router.Party) {

	party.Post("/get-signed-url", getSignedURLHandler)
}

func getSignedURLHandler(ctx context.Context) {

	signedURLRequest := &SignedURLRequest{}
	if err := ctx.ReadJSON(signedURLRequest); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.JSON(map[string]interface{}{"error": "Unable to read filename"})
		return
	}

	url, err := getUploadURL(signedURLRequest.Filename, signedURLRequest.MimeType)
	if err != nil {
		panic(err)
	}

	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(map[string]interface{}{"signedUrl": url})
}

func getUploadURL(filename string, mimeType string) (string, error) {
	req, _ := s3Service.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String("assets.muskokacabco.com"),
		Key:         aws.String("assets/uploads/" + filename),
		ContentType: aws.String(mimeType),
	})
	return req.Presign(15 * time.Minute)
}

func deleteS3Object(filename string) error {
	cleanFilename := strings.Replace(filename, "+", " ", 1)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String("assets.muskokacabco.com"),
		Key:    aws.String("assets/uploads/" + cleanFilename),
	}

	/*result*/
	_, err := s3Service.DeleteObject(input)
	return err
}
