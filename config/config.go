package config

func IsDev() bool {
	return Environment() == "development"
}

func IsProd() bool {
	return Environment() == "production"
}
