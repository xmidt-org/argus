// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/spf13/cast"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/bascule"
	"go.uber.org/fx"
)

// Exported access level default values which application code may want to use.
const (
	DefaultAccessLevelAttributeKey   = "access-level"
	DefaultAccessLevelAttributeValue = 0
)

// ElevatedAccessLevelAttributeValue is the value that will be used when a request
// passes all checks for running in elevated access mode.
const ElevatedAccessLevelAttributeValue = 1

// internal default values
const (
	defaultAccessLevelCapabilityName = "xmidt:svc:admin"
)

var defaultAccessLevelPath = []string{"capabilities"}

// AccessLevel provides logic for resolving the correct access level for a
// request given its bascule attributes.
type AccessLevel struct {
	Resolve      accessLevelResolver
	AttributeKey string
}

// accessLevelResolver lets users of accessLevelBearerTokenFactory determine what access level value is assigned to a
// request based on its capabilities.
type accessLevelResolver func(bascule.Attributes) int

type accessLevelCapabilitySource struct {
	// Name is the capability we will search for inside the capability list pointed by path.
	// If this value is found in the list, the access level assigned to the request will be 1. Otherwise, it will be 0.
	// (Optional) defaults to 'xmidt:svc:admin'
	Name string

	// Path is the list of nested keys to get to the claim which contains the capabilities.
	// (Optional) default: ["capabilities"]
	Path []string
}

type accessLevelConfig struct {
	AttributeKey     string
	CapabilitySource accessLevelCapabilitySource
}

func defaultAccessLevel() AccessLevel {
	return AccessLevel{
		AttributeKey: DefaultAccessLevelAttributeKey,
		Resolve: func(_ bascule.Attributes) int {
			return DefaultAccessLevelAttributeValue
		},
	}
}

func validateAccessLevelConfig(config *accessLevelConfig) {
	if len(config.AttributeKey) < 1 {
		config.AttributeKey = DefaultAccessLevelAttributeKey
	}

	if len(config.CapabilitySource.Name) < 1 {
		config.CapabilitySource.Name = defaultAccessLevelCapabilityName
	}

	if len(config.CapabilitySource.Path) < 1 {
		config.CapabilitySource.Path = defaultAccessLevelPath
	}
}

func newContainsAttributeAccessLevel(config *accessLevelConfig) AccessLevel {
	validateAccessLevelConfig(config)

	resolve := func(attributes bascule.Attributes) int {
		capabilitiesClaim, ok := bascule.GetNestedAttribute(attributes, config.CapabilitySource.Path...)
		if !ok {
			return DefaultAccessLevelAttributeValue
		}
		capabilities := cast.ToStringSlice(capabilitiesClaim)

		for _, capability := range capabilities {
			if capability == config.CapabilitySource.Name {
				return ElevatedAccessLevelAttributeValue
			}
		}

		return DefaultAccessLevelAttributeValue
	}

	return AccessLevel{
		AttributeKey: config.AttributeKey,
		Resolve:      resolve,
	}
}

func provideAccessLevel(key string) fx.Option {
	return fx.Options(
		fx.Provide(
			arrange.UnmarshalKey(key, &accessLevelConfig{}),
			func(c *accessLevelConfig) AccessLevel {
				if c == nil {
					return defaultAccessLevel()
				}
				return newContainsAttributeAccessLevel(c)
			},
		),
	)
}
