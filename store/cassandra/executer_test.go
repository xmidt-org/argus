package cassandra

//
// import (
// 	"github.com/gocql/gocql"
// 	"github.com/stretchr/testify/require"
// 	"github.com/xmidt-org/argus/store"
// 	"github.com/xmidt-org/argus/store/storetest"
// 	"github.com/xmidt-org/webpa-common/logging"
// 	"github.com/xmidt-org/webpa-common/xmetrics"
// 	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
// 	"testing"
// 	"time"
// )
//
// func TestCassandraStore(t *testing.T) {
// 	p := xmetricstest.NewProvider(nil, func() []xmetrics.Metric {
// 		return []xmetrics.Metric{
// 			{
// 				Name: PoolInUseConnectionsGauge,
// 				Type: "gauge",
// 				Help: " The number of connections currently in use",
// 			},
// 			{
// 				Name:       SQLDurationSeconds,
// 				Type:       "histogram",
// 				Help:       "A histogram of latencies for requests.",
// 				Buckets:    []float64{0.0625, 0.125, .25, .5, 1, 5, 10, 20, 40, 80, 160},
// 				LabelNames: []string{store.TypeLabel},
// 			},
// 			{
// 				Name:       SQLQuerySuccessCounter,
// 				Type:       "counter",
// 				Help:       "The total number of successful SQL queries",
// 				LabelNames: []string{store.TypeLabel},
// 			},
// 			{
// 				Name:       SQLQueryFailureCounter,
// 				Type:       "counter",
// 				Help:       "The total number of failed SQL queries",
// 				LabelNames: []string{store.TypeLabel},
// 			},
// 			{
// 				Name: SQLInsertedRecordsCounter,
// 				Type: "counter",
// 				Help: "The total number of rows inserted",
// 			},
// 			{
// 				Name: SQLReadRecordsCounter,
// 				Type: "counter",
// 				Help: "The total number of rows read",
// 			},
// 			{
// 				Name: SQLDeletedRecordsCounter,
// 				Type: "counter",
// 				Help: "The total number of rows deleted",
// 			},
// 		}
// 	})
// 	config := gocql.NewCluster("127.0.0.1")
// 	config.Keyspace = "config"
//
// 	client, err := connect(config, logging.NewTestLogger(nil, t))
// 	require.Empty(t, err)
//
// 	s := &CassandraClient{
// 		client: client,
// 		config: CassandraConfig{},
// 		logger: logging.NewTestLogger(nil, t),
// 		measures: Measures{
// 			PoolInUseConnections: p.NewGauge(PoolInUseConnectionsGauge),
// 			SQLDuration:          p.NewHistogram(SQLDurationSeconds, 11),
// 			SQLQuerySuccessCount: p.NewCounter(SQLQuerySuccessCounter),
// 			SQLQueryFailureCount: p.NewCounter(SQLQueryFailureCounter),
// 			SQLInsertedRecords:   p.NewCounter(SQLInsertedRecordsCounter),
// 			SQLReadRecords:       p.NewCounter(SQLReadRecordsCounter),
// 			SQLDeletedRecords:    p.NewCounter(SQLDeletedRecordsCounter),
// 		},
// 	}
// 	storetest.StoreTest(s, time.Duration(storetest.GenericTestKeyPair.TTL)*time.Second, t)
// }
