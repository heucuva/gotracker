package config

type Overlay interface {
	Wants(param any) bool
	Update(cfg, param any) error
}
