package command

import "fmt"

type Architecture string

func (p *Architecture) Set(value string) error {
	switch value {
	case "amd64", "arm64":
		*p = Architecture(value)
		return nil
	default:
		return fmt.Errorf("invalid architecture: %s", value)
	}
}

func (p *Architecture) String() string {
	return string(*p)
}
