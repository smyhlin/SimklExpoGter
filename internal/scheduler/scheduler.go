package scheduler

type Config struct {
	TaskName       string
	Description    string
	ExecutablePath string
	Arguments      []string
	Frequency      string
	Time           string
	Days           []string
}

type State struct {
	Supported  bool
	Installed  bool
	TaskName   string
	Status     string
	NextRunAt  string
	LastRunAt  string
	LastResult string
	Message    string
}

type Manager interface {
	Supported() bool
	Sync(Config) (State, error)
	Remove(string) error
	Query(string) (State, error)
}

func NewManager() Manager {
	return newManager()
}
