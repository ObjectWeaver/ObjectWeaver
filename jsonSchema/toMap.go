package jsonSchema

// ToMap converts the Definition struct to a map representation
func (d Definition) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if d.Type != "" {
		result["type"] = d.Type
	}
	if d.Instruction != "" {
		result["instruction"] = d.Instruction
	}
	if len(d.Properties) > 0 {
		propertiesMap := make(map[string]interface{})
		for key, value := range d.Properties {
			propertiesMap[key] = value.ToMap()
		}
		result["properties"] = propertiesMap
	}
	if d.Items != nil {
		result["items"] = d.Items.ToMap()
	}
	if d.Model != "" {
		result["model"] = d.Model
	}
	if len(d.ProcessingOrder) > 0 {
		result["processingOrder"] = d.ProcessingOrder
	}

	return result
}
