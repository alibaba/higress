HEADER := Content-Type: application/json


.PHONY: docker docker-down proxy cache weaviate-collection weaviate-obj

all: docker proxy cache weaviate-collection weaviate-obj

docker:
	docker compose up

docker-down:
	docker compose down

proxy:
	cd ../plugins/wasm-go/extensions/ai-proxy && \
	tinygo build -o ai-proxy.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100' . && \
	mv ai-proxy.wasm ../../../../docker-compose-test/


cache:
	cd ../plugins/wasm-go/extensions/ai-cache && \
	tinygo build -o ai-cache.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer proxy_wasm_version_0_2_100' . && \
	mv ai-cache.wasm ../../../../docker-compose-test/

weaviate-obj:
	curl --request POST --url http://localhost:8081/v1/objects -H "$(HEADER)" --data '{	"class": "",	"vectorWeights": {},	"properties": {},	"id": "",	"creationTimeUnix": 1,	"lastUpdateTimeUnix": 1,	"vector": [		1	],	"vectors": {		"ANY_ADDITIONAL_PROPERTY": [1]	},	"tenant": "",	"additional": {		"ANY_ADDITIONAL_PROPERTY": {}	}}'

weaviate-collection:
	curl -X POST http://localhost:8081/v1/schema -H "$(HEADER)" -d '{
	"class": "",
	"vectorConfig": {
		"<vector_name>": {
			"vectorizer": "none",
			"vectorIndexType": "hnsw",
			"vectorIndexConfig": {}
		}
	},
	"vectorIndexType": "hnsw",
	"vectorIndexConfig": {},
	"shardingConfig": {
		"desiredCount": 1,
		"virtualPerPhysical": 128
	},
	"replicationConfig": {
		"factor": 1,
		"asyncEnabled": true
	},
	"invertedIndexConfig": {
		"cleanupIntervalSeconds": 60,
		"bm25": {
			"k1": 1.2,
			"b": 0.75
		},
		"stopwords": {
			"preset": "en",
			"additions": [],
			"removals": []
		},
		"indexTimestamps": false,
		"indexNullState": false,
		"indexPropertyLength": false
	},
	"multiTenancyConfig": {
		"enabled": false,
		"autoTenantCreation": true,
		"autoTenantActivation": true
	},
	"vectorizer": "none",
	"moduleConfig": {
		"<module_name>": {
			"vectorizeClassName": true
		}
	},
	"description": "",
	"properties": [
		{
			"dataType": [
				"string"
			],
			"description": "",
			"moduleConfig": {
				"<module_name>": {
					"skip": false,
					"vectorizePropertyName": true
				}
			},
			"name": "",
			"indexInverted": true,
			"indexFilterable": true,
			"indexSearchable": true,
			"indexRangeFilters": true,
			"tokenization": "word",
			"nestedProperties": [
				{
					"name": "",
					"dataType": [
						null
					],
					"description": "",
					"indexFilterable": true,
					"indexSearchable": true,
					"indexRangeFilters": true,
					"tokenization": "word",
					"nestedProperties": [
						null
					]
				}
			]
		}
	]
	}'