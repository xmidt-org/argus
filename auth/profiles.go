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
	// ErrMissingTargetServer is returned whenever a bascule profile lists a targetServer that
	// does not exist.
	ErrMissingTargetServer = errors.New("invalid targetServer in bascule profile")

	// ErrUnusedProfile is returned when no targetServer was specified for a profile.
	ErrUnusedProfile = errors.New("profile has no targetServers")
)

// profilesIn is the parameter struct for generating bascule profiles
// from config.
type profilesIn struct {
	fx.In

	// Logger is the required go-kit logger that will receive health logging output.
	Logger log.Logger

	// Unmarshaller is the required configuration unmarshaller strategy.
	Unmarshaller config.Unmarshaller
}

// profilesOut is the result parameter containing bascule profiles.
type profilesOut struct {
	fx.Out

	// Profiles is a mapping from a target server name (i.e. 'servers.primary') to the profile that
	// should be used to generate its auth chain
	Profiles map[string]*Profile `name:"server_profiles"`
}

// profile is the struct to help read on bascule profle information from config.
type Profile struct {
	TargetServers   []string
	Basic           []string
	Bearer          jwtValidator
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

// UnmarshalProfiles returns an uber/fx provider that reads configuration from a Viper
// instance and initializes bascule profiles.
func UnmarshalProfiles(configKey string) func(profilesIn) (map[string]*Profile, error) {
	return func(in profilesIn) (map[string]*Profile, error) {
		var sourceProfiles []Profile
		servers := map[string]bool{"primary": true}
		fmt.Println("unmarshalling lol")

		if err := in.Unmarshaller.UnmarshalKey(configKey, &sourceProfiles); err != nil {
			return nil, fmt.Errorf("failed to unmarshal bascule profile config: %w", err)
		}

		if len(sourceProfiles) < 1 {
			in.Logger.Log(level.Key(), level.InfoValue(), xlog.MessageKey(), "No bascule profiles configured")
			return nil, ErrMissingTargetServer
		}

		profiles := make(map[string]*Profile)

		for _, sourceProfile := range sourceProfiles {
			if len(sourceProfile.TargetServers) < 1 {
				return nil, ErrUnusedProfile
			}

			for _, targetServer := range sourceProfile.TargetServers {
				if !servers[targetServer] {
					in.Logger.Log(level.Key(), level.ErrorValue(), xlog.MessageKey(), "Bascule profile targetServer does not exist.", "targetServer", targetServer)
					return nil, ErrMissingTargetServer
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

type ProfileProvider struct {
	ServerName string
}

func (p ProfileProvider) Provide(profiles map[string]*Profile) *Profile {
	fmt.Println("Provider the Primary profile")
	return profiles[p.ServerName]
}

func (p ProfileProvider) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   "primary_profile",
		Target: p.Provide,
	}
}
