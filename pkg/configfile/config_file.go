package configfile

import (
	"errors"
	"strings"

	"github.com/go-ini/ini"
)

type Meta struct {
	Path    string
	Cfg     *ini.File
	Section *ini.Section
}

func Load(path string) (*Meta, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowNonUniqueSections: true, AllowDuplicateShadowValues: true}, path)
	if err != nil {
		return nil, err
	}

	return &Meta{
		Path: path,
		Cfg:  cfg,
	}, nil
}

func (m *Meta) Save() error {
	return m.Cfg.SaveTo(m.Path)
}

func ParseKeyFromSectionString(path string, section string, key string) (string, error) {
	c, err := Load(path)
	if err != nil {
		return "", err
	}

	v := c.Cfg.Section(section).Key(key).String()
	if v == "" {
		return "", errors.New("not found")
	}

	return v, nil
}

func (m *Meta) SetKeySectionString(section string, key string, value string) {
	m.Cfg.Section(section).Key(key).SetValue(strings.ToLower(value))
}

func (m *Meta) NewSection(section string) error {
	s, err := m.Cfg.NewSection(section)
	if err != nil {
		return err
	}
	m.Section = s
	return nil
}

func (m *Meta) SetKeyToNewSectionString(key string, value string){
	m.Section.Key(key).SetValue(strings.ToLower(value))
}

func MapTo(cfg *ini.File, section string, v interface{}) error {
	if err := cfg.Section(section).MapTo(v); err != nil {
		return err
	}

	return nil
}