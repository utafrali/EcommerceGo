module github.com/utafrali/EcommerceGo/services/gateway

go 1.23

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/stretchr/testify v1.9.0
	github.com/utafrali/EcommerceGo/pkg v0.0.0
	golang.org/x/time v0.5.0
)

require (
	github.com/caarlos0/env/v10 v10.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/utafrali/EcommerceGo/pkg => ../../pkg
