module github.com/reoring/goskema/examples/http_echo

go 1.25.1

require (
	github.com/labstack/echo/v4 v4.13.4
	github.com/reoring/goskema v0.0.0
	github.com/reoring/goskema/middleware/echo v0.0.0
)

require (
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)

replace github.com/reoring/goskema => ../..

replace github.com/reoring/goskema/middleware/echo => ../../middleware/echo
