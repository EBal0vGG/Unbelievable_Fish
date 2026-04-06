package catalog

type Fish struct {
	fishID      string
	name        string
	description string
}

func NewFish(fishID, name, description string) (*Fish, error) {
	if isBlank(fishID) || isBlank(name) {
		return nil, ErrInvalidIdentifier
	}

	return &Fish{
		fishID:      fishID,
		name:        name,
		description: description,
	}, nil
}

func (f *Fish) ID() string          { return f.fishID }
func (f *Fish) Name() string        { return f.name }
func (f *Fish) Description() string { return f.description }

func (f *Fish) Update(name, description string) error {
	if isBlank(name) {
		return ErrInvalidIdentifier
	}
	f.name = name
	f.description = description
	return nil
}
