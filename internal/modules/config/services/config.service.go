package config

import (
	"github.com/99designs/keyring"
	"github.com/PedroHercules/gommit/internal/modules/config/entities"
)

const (
	ServiceName = "gommit"
	LlmKeyName  = "llm_key"
)

type ConfigService struct {
	keyring keyring.Keyring
	config  *entities.ConfigEntity
}

func NewConfigService() (*ConfigService, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: ServiceName,
	})
	if err != nil {
		return nil, err
	}

	config := &entities.ConfigEntity{}
	err = config.Load()
	if err != nil {
		return nil, err
	}

	return &ConfigService{
		keyring: ring,
		config:  config,
	}, nil
}

func (s *ConfigService) SetLlmKey(key string) error {
	return s.keyring.Set(keyring.Item{
		Key:  LlmKeyName,
		Data: []byte(key),
	})
}

func (s *ConfigService) GetLlmKey() (string, error) {
	item, err := s.keyring.Get(LlmKeyName)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (s *ConfigService) RemoveLlmKey() error {
	return s.keyring.Remove(LlmKeyName)
}

func (s *ConfigService) GetConfig() *entities.ConfigEntity {
	return s.config
}

func (s *ConfigService) SaveConfig() error {
	return s.config.Save()
}

func (s *ConfigService) SetDefaultModel(model string) error {
	s.config.DefaultModel = model
	return s.SaveConfig()
}

func (s *ConfigService) GetDefaultModel() string {
	return s.config.DefaultModel
}

func (s *ConfigService) RemoveDefaultModel() error {
	s.config.DefaultModel = ""
	return s.SaveConfig()
}