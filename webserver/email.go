package muskoka

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

type Email struct {
	To      []string
	From    string
	Subject string
	Text    string
}

var sesService *ses.SES

func InitSES() {
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	sesService = ses.New(sess, aws.NewConfig().WithRegion("us-east-1"))
}

func (email *Email) Send() error {

	stringArrayToAWS(email.To)
	params := &ses.SendEmailInput{
		Destination: &ses.Destination{ // Required
			ToAddresses: stringArrayToAWS(email.To[:]),
		},
		Message: &ses.Message{ // Required
			Body: &ses.Body{ // Required
				Text: &ses.Content{
					Data: aws.String(email.Text), // Required
				},
			},
			Subject: &ses.Content{ // Required
				Data: aws.String(email.Subject), // Required
			},
		},
		Source: aws.String(email.From), // Required
		ReplyToAddresses: []*string{
			aws.String(email.From), // Required
			// More values...
		},
	}
	_, err := sesService.SendEmail(params)
	return err
}

func stringArrayToAWS(arr []string) []*string {
	var awsArr = make([]*string, len(arr))
	for i, element := range arr {
		awsArr[i] = aws.String(element)
	}
	return awsArr
}
