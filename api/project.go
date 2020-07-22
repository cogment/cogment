package api

type ProjectConfig struct {
	Import       *Import
	Commands     map[string]string
	Trial        *Trial
	Environment  *Environment
	ActorClasses []*ActorClass `yaml:"actor_classes"`
	TrialParams  *TrialParams  `yaml:"trial_params"`
	ProjectName	 string
	CliVersion	 string
}

type Environment struct {
	ConfigType string `yaml:"config_type"`
}

type Import struct {
	Proto      []string
	ProtoAlias []string
	Python     []string
	Javascript []string
}

type Trial struct {
	ConfigType string   `yaml:"config_type"`
	PreHooks   []string `yaml:"pre_hooks"`
}

type Observation struct {
	Space        string
	Delta        string
	DeltaApplyFn *DeltaApplyFn `yaml:"delta_apply_fn"`
}

type DeltaApplyFn struct {
	Python     string
	Javascript string
}

type ActorClass struct {
	Id     string
	Action struct {
		Space string
	}
	Observation *Observation
	ConfigType  string `yaml:"config_type"`
}

type TrialParams struct {
	Environment struct {
		Endpoint string
		Config   map[string]interface{}
	}
	Actors []*Actor
}

type Actor struct {
	ActorClass string `yaml:"actor_class"`
	Endpoint   string
	Config     map[string]interface{}
}

func (p *ProjectConfig) CountActorsByActorClass(id string) (countAi, countHuman int) {
	countAi = 0
	countHuman = 0

	for _, actor := range p.TrialParams.Actors {
		if actor.ActorClass != id {
			continue
		}

		if actor.Endpoint == "human" {
			countHuman += 1
		} else {
			countAi += 1
		}
	}

	return countAi, countHuman
}

func (p *ProjectConfig) HasAiByActorClass(id string) bool {
	countAi, _ := p.CountActorsByActorClass(id)
	return countAi > 0
}

func (p *ProjectConfig) HasHumanByActorClass(id string) bool {
	_, countHuman := p.CountActorsByActorClass(id)
	return countHuman > 0
}
