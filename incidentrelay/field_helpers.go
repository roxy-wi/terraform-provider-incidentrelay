package incidentrelay

func reqString(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindString, Required: true, Description: description}
}

func reqStringForceNew(name, description string) fieldDef {
	field := reqString(name, description)
	field.ForceNew = true
	return field
}

func optString(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindString, Optional: true, Description: description}
}

func optStringForceNew(name, description string) fieldDef {
	field := optString(name, description)
	field.ForceNew = true
	return field
}

func optStringDefault(name, defaultValue, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindString, Optional: true, Default: defaultValue, Description: description}
}

func computedString(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindString, Computed: true, Description: description}
}

func reqInt(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindInt, Required: true, Description: description}
}

func optInt(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindInt, Optional: true, Description: description}
}

func optIntDefault(name string, defaultValue int, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindInt, Optional: true, Default: defaultValue, Description: description}
}

func computedInt(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindInt, Computed: true, Description: description}
}

func optBoolDefault(name string, defaultValue bool, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindBool, Optional: true, Default: defaultValue, Description: description}
}

func computedBool(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindBool, Computed: true, Description: description}
}

func optJSONDefault(name, apiName, defaultValue, description string) fieldDef {
	return fieldDef{Name: name, APIName: apiName, Kind: kindJSON, Optional: true, Default: defaultValue, Description: description}
}

func reqJSON(name, apiName, description string) fieldDef {
	return fieldDef{Name: name, APIName: apiName, Kind: kindJSON, Required: true, Description: description}
}

func computedJSON(name, apiName, description string) fieldDef {
	return fieldDef{Name: name, APIName: apiName, Kind: kindJSON, Computed: true, Description: description}
}

func optStringSet(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindStringSet, Optional: true, Description: description}
}

func optIntSet(name, description string) fieldDef {
	return fieldDef{Name: name, Kind: kindIntSet, Optional: true, Description: description}
}

func reqIntForceNew(name, description string) fieldDef {
	field := reqInt(name, description)
	field.ForceNew = true
	return field
}

func optSensitiveString(name, description string) fieldDef {
	field := optString(name, description)
	field.Sensitive = true
	return field
}

func computedSensitiveString(name, description string) fieldDef {
	field := computedString(name, description)
	field.Sensitive = true
	return field
}
