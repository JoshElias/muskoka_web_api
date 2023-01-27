package muskoka

import (
	"github.com/kataras/iris"
	"github.com/rs/cors"
)

func CreateApp() {

	app := iris.New()
	//app.UseFunc(SecureMiddleware)
	//app.Adapt(httprouter.New())
	//app.Adapt(iris.DevLogger())

	corsOptions := cors.Options{
		AllowedMethods:   []string{"OPTIONS", "GET", "PUT", "POST", "DELETE"},
		AllowCredentials: true,
		AllowedOrigins:   []string{"*"},
		AllowedHeaders:   []string{"X-Requested-With", "Content-Type"},
	}
	corsWrapper := cors.New(corsOptions).ServeHTTP
	app.WrapRouter(corsWrapper)

	authAPI := app.Party("/auth")
	{
		authAPI.Post("/register", RegisterHandler)
		authAPI.Post("/verify", VerifyHandler)
		authAPI.Post("/login", LoginHandler)
	}

	CreateUploadAPI(app.Party("/upload"))
	CreateColourAPI(app.Party("/colour"))
	CreateDoorSampleAPI(app.Party("/door-sample"))
	CreateDoorStyleAPI(app.Party("/door-style"))
	CreateDoorStyleTypeAPI(app.Party("/door-style-type"))
	CreateGallerySampleAPI(app.Party("/gallery-sample"))
	CreateImageTypeAPI(app.Party("/image-type"))
	CreateWoodAPI(app.Party("/wood"))
	CreateDealerAPI(app.Party("/dealer"))

	app.Run(iris.Addr(":8080"), iris.WithoutVersionChecker)

	defer CloseDb()
}
