package config

func GetLlmKey() (string, error) {
	configService, err := NewConfigService()
	if err != nil {
		return "", err
	}

	return configService.GetLlmKey()
}