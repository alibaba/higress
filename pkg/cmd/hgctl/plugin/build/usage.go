package plugin

import "fmt"

type WasmUsage struct {
	I18nType      I18nType
	Description   string
	ConfigEntries []ConfigEntry
	Example       string
}

type ConfigEntry struct {
	Name        string
	Type        string
	Requirement string
	Default     string
	Description string
}

func GetUsageFromMeta(meta *WasmPluginMeta) []*WasmUsage {
	usages := make([]*WasmUsage, 0)
	for i18n, desc := range meta.Info.XDescriptionI18n {
		u := WasmUsage{
			I18nType:      i18n,
			Description:   desc,
			ConfigEntries: make([]ConfigEntry, 0),
			Example:       "",
		}
		getConfigEntryFromSchema(meta.Spec.ConfigSchema.OpenAPIV3Schema, &u.ConfigEntries, "", "", i18n, false)
		usages = append(usages, &u)
	}

	return usages
}

func getConfigEntryFromSchema(schema *JSONSchemaProps, entries *[]ConfigEntry, parent, name string, i18n I18nType, required bool) {
	newName := constructName(parent, name)
	reqs := schema.HandleRequirements(required)

	switch schema.Type {
	case "object":
		for name, props := range schema.Properties {
			required = schema.IsRequired(name)
			getConfigEntryFromSchema(&props, entries, newName, name, i18n, required)
		}
	case "array":
		itemType := schema.Items.Schema.Type
		e := ConfigEntry{
			Name:        newName,
			Type:        fmt.Sprintf("array of %s", itemType),
			Requirement: RequirementsJoinByI18n(reqs, i18n),
			Default:     schema.GetDefaultValue(),
			Description: schema.XDescriptionI18n[i18n],
		}
		*entries = append(*entries, e)
		if itemType == "object" {
			getConfigEntryFromSchema(schema.Items.Schema, entries, fmt.Sprintf("%s[*]", newName), "", i18n, false)
		}
	default:
		e := ConfigEntry{
			Name:        newName,
			Type:        schema.Type,
			Requirement: RequirementsJoinByI18n(reqs, i18n),
			Default:     schema.GetDefaultValue(),
			Description: schema.XDescriptionI18n[i18n],
		}
		*entries = append(*entries, e)
	}
}

func constructName(parent, name string) string {
	newName := name
	if parent != "" {
		if name != "" {
			newName = fmt.Sprintf("%s.%s", parent, name)
		} else {
			newName = parent
		}
	}
	return newName
}
