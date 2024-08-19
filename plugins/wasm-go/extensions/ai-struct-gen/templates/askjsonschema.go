package templates

var AskJsonSchemaTemp = `{
	"type": "json_schema",
	"json_schema": {
		"name": "generateJsonSchemaByRequst",
		"strict": true,
        "schema": {
            "type": "object",
            "properties": {
                "json": {
                    "type": "string",
                    "description": "Json schema, do not involve any other description"
                },
                "reason": {
                    "type": "string",
                    "description": "Description"
                }
            },
            "required": [
                "json",
                "reason"
            ],
            "additionalProperties": false
        }
    }
}`
