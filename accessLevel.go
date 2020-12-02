package main

import (
	"fmt"

	"github.com/spf13/cast"
	"github.com/xmidt-org/argus/store"
	"github.com/xmidt-org/bascule"
)

// Access level default values.
const (
	DefaultAccessLevelAttributeKey   = "access-level"
	DefaultAccessLevelAttributeValue = 0
)

type AccessLevelConfig struct {
	AttributeKey string
	Resolver     AccessLevelResolver
}

type superUserAccessLevelConfig struct {
	CapabilityListClaimPath []string
	SuperUserCapability     string
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

func (s superUserAccessLevelConfig) GetCapabilityListClaimPath() []string {
	if len(s.CapabilityListClaimPath) > 0 {
		return s.CapabilityListClaimPath
	}
	return []string{"capabilities"}
}

// AccessLevelResolver is the function signature to be implemented by users of the access level token factory.
type AccessLevelResolver func(bascule.Attributes) int

func superUserAccessLevelResolver(cfg superUserAccessLevelConfig) AccessLevelResolver {
	return func(attributes bascule.Attributes) int {
		capabilitiesClaim, ok := bascule.GetNestedAttribute(attributes, cfg.GetCapabilityListClaimPath()...)
		fmt.Printf("capabilities claim=%v, capability=%v \n", capabilitiesClaim, cfg.SuperUserCapability)
		if !ok {
			return DefaultAccessLevelAttributeValue
		}
		capabilities := cast.ToStringSlice(capabilitiesClaim)

		for _, capability := range capabilities {
			if capability == cfg.SuperUserCapability {
				return store.SuperUserAccessLevel
			}
		}

		return DefaultAccessLevelAttributeValue
	}
}
