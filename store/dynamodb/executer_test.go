package dynamodb

// import (
// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/stretchr/testify/require"
// 	"github.com/xmidt-org/argus/store/test"
// 	"github.com/xmidt-org/webpa-common/logging"
// 	"testing"
// 	"time"
// )
//
// func TestDynamoDB(t *testing.T) {
// 	require := require.New(t)
//
// 	client, err:= createDynamoDBexecutor(aws.Config{
// 		Endpoint:                          aws.String("http://localhost:8042"),
// 		Region:                            aws.String("us-east-2"),
// 	}, "", "demo-config", logging.NewTestLogger(nil, t))
// 	require.NoError(err)
//
// 	test.StoreTest(client, time.Duration(test.GenericTestKeyPair.TTL)*time.Second, t)
// }
