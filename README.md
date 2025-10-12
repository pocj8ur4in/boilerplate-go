# [pocj8ur4in/boilerplate-go](https://github.com/pocj8ur4in/boilerplate-go)

[![CI](https://github.com/pocj8ur4in/boilerplate-go/workflows/CI/badge.svg)](https://github.com/pocj8ur4in/boilerplate-go/actions)
[![codecov](https://codecov.io/gh/pocj8ur4in/boilerplate-go/branch/main/graph/badge.svg)](https://codecov.io/gh/pocj8ur4in/boilerplate-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/pocj8ur4in/boilerplate-go)](https://goreportcard.com/report/github.com/pocj8ur4in/boilerplate-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pocj8ur4in/boilerplate-go)](https://github.com/pocj8ur4in/boilerplate-go)
[![License](https://img.shields.io/github/license/pocj8ur4in/boilerplate-go)](https://github.com/pocj8ur4in/boilerplate-go/blob/main/LICENSE)

boilerplate code for http api server on go with stdlib, [chi](https://github.com/go-chi/chi), [pgx](https://github.com/jackc/pgx), [go-redis](https://github.com/redis/go-redis), [zerolog](https://github.com/rs/zerolog), [oapi-codegen](https://github.com/deepmap/oapi-codegen), [fx](https://github.com/uber-go/fx), and [sqlc](https://github.com/sqlc-dev/sqlc)

## How to install

1. clone repository
2. run `make rename` to rename repository and project names
    - repository name : `pocj8ur4in/boilerplate-go` -> `your-username/your-repository-name`
    - project name : `boilerplate` -> `your-project-name`
3. change names of directories and files
   - `internal/app/boilerplate` -> `internal/app/your-project-name`
   - `cmd/boilerplate` -> `cmd/your-project-name`
   - `grafana/provisioning/dashboards/boilerplate-dashboard.json` -> `grafana/provisioning/dashboards/your-project-name-dashboard.json`
4. run `make prepare` to update dependencies and continue setup
5. create `config.json` file by copying `config.example.json` and changing the values
6. add github actions secrets on your github repository
   - `CODECOV_TOKEN`: for codecov
7. register your repository on [codecov](https://codecov.io/)
8. push to your github repository

## How to run

1. run `make docker dev` to run the application in development mode with docker compose
2. run `make go test` to run the tests (some tests require docker to be running)
3. run `make go build` to build the application
4. run `make go run` to run the application

## How to contribute

1. fork this repository
2. create a new branch
3. make your changes
4. run `make go lint` to run linter (golangci-lint)
5. run `make go fmt` to run formatter (golangci-lint)
6. run `make go sec` to run security scan (gosec)
7. run `make openapi generate` to generate OpenAPI spec (openapi spec files in /api directory)
8. run `make sqlc generate` to generate SQL code (sql files in /sql directory)
9. create a pull request and check if github actions are passing
10. wait for the pull request to be merged

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
