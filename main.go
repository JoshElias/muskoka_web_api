package main

import "bitbucket.com/daemontech/muskoka-web-api/webserver"

func main() {
	muskoka.GetDBConnection()
	defer muskoka.CloseDb()

	muskoka.InitSES()
	muskoka.InitS3()

	muskoka.InitColour()
	muskoka.InitWood()
	muskoka.InitDoorStyleType()
	muskoka.InitDoorStyle()
	muskoka.InitDoorStyleDoorStyleType()
	muskoka.InitImageType()
	muskoka.InitDoorSample()
	muskoka.InitGallerySample()
	muskoka.InitImage()
	muskoka.InitDealer()
	muskoka.InitUser()

	muskoka.CreateApp()
}
