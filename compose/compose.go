package compose

type PortDefinition struct {
	Port   uint16
	Public bool
}

type XCogment struct {
	Image       string
	Command     string
	Entrypoint  string
	Ports       []*PortDefinition
	Environment map[string]string
}

type Service struct {
	Image    string
	Build    interface{}
	XCogment *XCogment `yaml:"x-cogment"`
}

type Manifest struct {
	//Version  string
	Services map[string]Service
}
