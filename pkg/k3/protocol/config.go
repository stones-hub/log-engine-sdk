package protocol

// TODO 需要考虑ELK的真实的配置需要哪些，目前只写了一些
type ELKConfig struct {
	Address  []string
	Username string
	Password string
	Apikey   string
}
