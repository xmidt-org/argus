package auth

import (
	"errors"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/key"
	"github.com/xmidt-org/themis/config"
	"github.com/xmidt-org/themis/xlog"
	"go.uber.org/fx"
)

var (
	// ErrUnsupportedTargetServer is returned when a bascule profile lists a targetServer that
	// does not exist (not supported).
	ErrUnsupportedTargetServer = errors.New("unsupported targetServer in bascule profile")

	// ErrUnusedProfile means no targetServer was specified for a profile.
	ErrUnusedProfile = errors.New("profile has no targetServers")
)

// profilesIn is the parameter struct for parsing bascule profiles from config.
type profilesProviderIn struct {
	fx.In

	// Logger is the required go-kit logger that will receive logging output.
	Logger log.Logger

	// Unmarshaller is the required configuration unmarshaller strategy.
	Unmarshaller config.Unmarshaller
}

type profilesIn struct {
	fx.In
	Logger log.Logger

	Profiles map[string]*profile `name:"bascule_profiles" optional:"true"`
}

// profile is the struct to help read on bascule profle information from config.
type profile struct {
	TargetServers   []string
	Basic           []string
	Bearer          *jwtValidator
	CapabilityCheck capabilityValidatorConfig
}

// jwtValidator provides a convenient way to define jwt validator through config files.
type jwtValidator struct {
	// JWTKeys is used to create the key.Resolver for JWT verification keys.
	Keys key.ResolverFactory `json:"keys"`

	// Leeway is used to set the amount of time buffer should be given to JWT
	// time values, such as nbf.
	Leeway bascule.Leeway
}

// ProfilesUnmarshaler provides a component that reads configuration from a Viper instance and initializes
// bascule profiles. For now, bascule profiles should be optional. The configKey is the key we should unmarshal
// the profiles from and the supportedServers should include all servers that profiles could target.
type ProfilesUnmarshaler struct {
	Name             string
	ConfigKey        string
	SupportedServers []string
}

func (p ProfilesUnmarshaler) Unmarshal(configKey string, supportedServers ...string) func(in profilesProviderIn) (map[string]*profile, error) {
	servers := make(map[string]bool)
	for _, supportedServer := range supportedServers {
		servers[supportedServer] = true
	}
	return func(in profilesProviderIn) (map[string]*profile, error) {
		in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "unmarshaling bascule profiles")

		var sourceProfiles []profile
		if err := in.Unmarshaller.UnmarshalKey(configKey, &sourceProfiles); err != nil {
			return nil, fmt.Errorf("failed to unmarshal bascule profiles from config: %w", err)
		}

		if len(sourceProfiles) < 1 {
			in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "No bascule profiles configured.")
			return nil, nil
		}

		profiles := make(map[string]*profile)
		for _, sourceProfile := range sourceProfiles {
			if len(sourceProfile.TargetServers) < 1 {
				return nil, ErrUnusedProfile
			}

			for _, targetServer := range sourceProfile.TargetServers {
				if !servers[targetServer] {
					in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "Bascule profile targetServer does not exist.", "targetServer", targetServer)
					return nil, ErrUnsupportedTargetServer
				}

				if _, ok := profiles[targetServer]; ok {
					in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "A previous Bascule profile was used for this server. Skipping.", "targetServer", targetServer)
					continue
				}
				profiles[targetServer] = &sourceProfile
			}
		}

		return profiles, nil
	}
}

func (p ProfilesUnmarshaler) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   "bascule_profiles",
		Target: p.Unmarshal(p.ConfigKey, p.SupportedServers...),
	}
}

type profileProvider struct {
	serverName string
}

func (p profileProvider) Provide(in profilesIn) *profile {
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "returning profile", "server", p.serverName)
	return in.Profiles[p.serverName]
}

func (p profileProvider) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_profile", p.serverName),
		Target: p.Provide,
	}
}
