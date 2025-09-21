module config-manager-sample

go 1.25.1

replace github.com/reoring/goskema => ../..

require (
	github.com/reoring/goskema v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)
