# Integration Tests

## Startup
```bash
docker-compose up -d
docker exec -it yb-tserver-n1 /home/yugabyte/bin/cqlsh -f /create_db.cql
```

- you can double check the db is up with `cqlsh` then `select * from config.config;`
```bash
curl -X POST \
  'http://localhost:6600/store/hi/world?attributes=beta,stage,neat,123' \
  -H 'Content-Type: application/json' \
  -d '{
	"neato": "worldhi",
	"yeet" : {
		"magic" :[
			1,2,3
			],
		"cool":"hi"
	}
}'

curl -X GET \
  http://localhost:6600/store/hi
# get data

curl -X GET \
  'http://localhost:6600/store/hi?attributes=beta,stage,neat,123'

# will also get data
curl -X GET \
  'http://localhost:6600/store/hi?attributes=beta,stage,neat,123,a,s'
# returns no data
```