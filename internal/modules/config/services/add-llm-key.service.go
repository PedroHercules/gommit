package config

func AddLlmKey(key string) error {
	configService, err := NewConfigService()
	if err != nil {
		return err
	}

	return configService.SetLlmKey(key)
}
