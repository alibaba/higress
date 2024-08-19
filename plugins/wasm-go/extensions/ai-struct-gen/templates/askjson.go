package templates

var AskJsonTemp = `{
    "type": "json_schema",
    "json_schema": {
        "name": "generateAPIByRequst",
        "strict": true,
        "schema": {
            "type": "object",
            "properties": {
                "json": {
                    "type": "object",
                    "properties": {
                        "api_version": {
                            "type": "string",
                            "description": "API version"
                        },
                        "endpoint": {
                            "type": "string",
                            "description": "API endpoint"
                        },
                        "method": {
                            "type": "string",
                            "description": "API method"
                        },
                        "parameters": {
                            "type": "array",
                            "items": {
                                "type": "object",
                                "properties": {
                                    "name": {
                                        "type": "string",
                                        "description": "Parameter name"
                                    },
                                    "type": {
                                        "type": "string",
                                        "description": "Parameter type"
                                    }
                                },
                                "required": [
                                    "name",
                                    "type"
                                ],
                                "additionalProperties": false
                            }
                        },
                        "response": {
                            "type": "object",
                            "properties": {
                                "status": {
                                    "type": "string",
                                    "description": "Response status"
                                },
                                "message": {
                                    "type": "string",
                                    "description": "Response message"
                                }
                            },
                            "required": [
                                "status",
                                "message"
                            ],
                            "additionalProperties": false
                        }
                    },
                    "required": [
                        "api_version",
                        "endpoint",
                        "method",
                        "parameters",
                        "response"
                    ],
                    "additionalProperties": false
                },
                "reason": {
                    "type": "string",
                    "description": "Reason for the generated schema"
                },
                "listOfCases": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "caseName": {
                                "type": "string"
                            },
                            "caseDescription": {
                                "type": "string"
                            },
                            "input": {
                                "type": "string"
                            },
                            "output": {
                                "type": "string"
                            }
                        },
                        "required": [
                            "caseName",
                            "caseDescription",
                            "input",
                            "output"
                        ],
                        "additionalProperties": false
                    },
                    "description": "List of cases"
                }
            },
            "required": [
                "json",
                "reason",
                "listOfCases"
            ],
            "additionalProperties": false,
            "description": "API json for generating schema by request"
        }
    }
}
`
