```bash
docker-compose -f docker-compose-dynamodb-local.yaml up -d

```

```bash
aws dynamodb  --endpoint-url http://localhost:8042 create-table \
    --table-name demo-config \
    --attribute-definitions \
        AttributeName=bucket,AttributeType=S \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=bucket,KeyType=HASH \
        AttributeName=id,KeyType=RANGE \
    --provisioned-throughput \
        ReadCapacityUnits=10,WriteCapacityUnits=5 \
    --stream-specification StreamEnabled=true,StreamViewType=NEW_AND_OLD_IMAGES \
    --region us-east-2

aws dynamodb  --endpoint-url http://localhost:8042 --region us-east-2 update-time-to-live --table-name demo-config --time-to-live-specification "Enabled=true, AttributeName=expires"



aws dynamodb  --endpoint-url http://localhost:8042 update-table --table-name demo-config --cli-input-json  \
'{
  "ReplicaUpdates":
  [
    {
      "Create": {
        "RegionName": "us-east-1"
      }
    }
  ]
}'

aws dynamodb  --endpoint-url http://localhost:8042 update-table --table-name demo-config --cli-input-json  \
'{
  "ReplicaUpdates":
  [
    {
      "Create": {
        "RegionName": "us-west-1"
      }
    }
  ]
}'

```
