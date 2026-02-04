package semantix

//go:generate go tool cfgx generate -i config/config.toml -o config/config.gen.go --mode getter
//go:generate go tool sqlc generate
