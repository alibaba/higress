package templates

var AskVerifyTemp = `{
    "type": "json_schema",
    "json_schema": {
        "name": "verifyJsonByJsonSchema",
        "strict": true,
        "schema":{
            "type": "object",
            "properties": {
                "reason": {
                    "type": "string",
                    "description": "Reason for the unmatach between the JsonCase and the corresponding JsonSchema, and how to fix it"
                },
                "json": {
                    "type": "string",
                    "description": "Fixed JsonCase"
                }
            },
            "required": [
                "reason",
                "json"
            ],
            "additionalProperties": false
        }
    }
}`
