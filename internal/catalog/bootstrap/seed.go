package bootstrap

import _ "embed"

//go:embed fish_seed.json
var defaultFishSeedJSON []byte

func DefaultFishSeedJSON() []byte {
	return defaultFishSeedJSON
}
