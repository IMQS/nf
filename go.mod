module github.com/IMQS/nf

replace github.com/IMQS/serviceauth => ../serviceauth

go 1.13

require (
	github.com/BurntSushi/migration v0.0.0-20140125045755-c45b897f1335
	github.com/IMQS/gowinsvc v0.0.0-20171019081213-88eed8ddfe95
	github.com/IMQS/log v1.0.0
	github.com/IMQS/serviceauth v0.0.0-00010101000000-000000000000
	github.com/jinzhu/gorm v1.9.11
	github.com/julienschmidt/httprouter v1.3.0
)
