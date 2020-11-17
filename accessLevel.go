package main

import "github.com/xmidt-org/bascule"

// Access level default values.
const (
	DefaultAccessLevelAttributeKey   = "access-level"
	DefaultAccessLevelAttributeValue = 0
)

// Argus specific access level values
const (
	RegularUserAccessLevel int = iota
	SuperUserAccessLevel
)

type AccessLevelConfig struct {
	AttributeKey string
	Resolver     AccessLevelResolver
}

func (a AccessLevelConfig) GetAttributeKey() string {
	if a.AttributeKey != "" {
		return a.AttributeKey
	}
	return DefaultAccessLevelAttributeKey
}

func (a AccessLevelConfig) GetResolver() AccessLevelResolver {
	if a.Resolver != nil {
		return a.Resolver
	}

	return func(_ bascule.Attributes) int {
		return DefaultAccessLevelAttributeValue
	}
}

type superUserAccessLevelConfig struct {
	CapabilityListClaimPath []string
	SuperUserCapability     string
}

func (s superUserAccessLevelConfig) GetCapabilityListClaimPath() []string {
	if len(s.CapabilityListClaimPath) > 0 {
		return s.CapabilityListClaimPath
	}
	return []string{"capabilities"}
}

// AccessLevelResolver is the function left to the user of the tokenBearer
type AccessLevelResolver func(bascule.Attributes) int

func superUserAccessLevelResolver(cfg superUserAccessLevelConfig) AccessLevelResolver {
	return func(attributes bascule.Attributes) int {
		capabilitiesClaim, ok := bascule.GetNestedAttribute(attributes, cfg.GetCapabilityListClaimPath()...)
		if !ok {
			return RegularUserAccessLevel
		}
		capabilities, ok := capabilitiesClaim.([]string)
		if !ok {
			return RegularUserAccessLevel
		}

		for _, capability := range capabilities {
			if capability == cfg.SuperUserCapability {
				return SuperUserAccessLevel

			}
		}

		return RegularUserAccessLevel
	}

}
