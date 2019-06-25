package outputelastic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/olivere/elastic"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsaikd/gogstash/config"
	"github.com/tsaikd/gogstash/config/goglog"
	"github.com/tsaikd/gogstash/config/logevent"
)

func init() {
	goglog.Logger.SetLevel(logrus.DebugLevel)
	config.RegistOutputHandler(ModuleName, InitHandler)
}

func Test_SSLCertValidation(t *testing.T) {
	assert := assert.New(t)
	// check default config is 'true'
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()
	ts.StartTLS()

	conf, err := config.LoadFromYAML([]byte(strings.TrimSpace(`
debugch: true
output:
  - type: elastic
    url: ["` + ts.URL + `"]
    index: "gogstash-index-test"
    document_type: "testtype"
    document_id: "%{fieldstring}"
    bulk_actions: 0
	`)))
	assert.Nil(err)
	assert.NotNil(conf)
	_, err = InitHandler(context.TODO(), &conf.OutputRaw[0])
	// expect error not nil as certificate is not trusted by default
	assert.NotNil(err)

	conf, err = config.LoadFromYAML([]byte(strings.TrimSpace(`
debugch: true
output:
  - type: elastic
    url: ["` + ts.URL + `"]
    index: "gogstash-index-test"
    document_type: "testtype"
    document_id: "%{fieldstring}"
    bulk_actions: 0
    ssl_certificate_validation: true
	`)))
	assert.Nil(err)
	assert.NotNil(conf)
	_, err = InitHandler(context.TODO(), &conf.OutputRaw[0])
	// again expect error not nil as certificate is not trusted and we requested ssl_certificate_validation
	assert.NotNil(err)

	conf, err = config.LoadFromYAML([]byte(strings.TrimSpace(`
debugch: true
output:
  - type: elastic
    url: ["` + ts.URL + `"]
    index: "gogstash-index-test"
    document_type: "testtype"
    document_id: "%{fieldstring}"
    bulk_actions: 0
    ssl_certificate_validation: false
	`)))
	assert.Nil(err)
	assert.NotNil(conf)
	_, err = InitHandler(context.TODO(), &conf.OutputRaw[0])
	// expect no error this time as ssl_certificate_validation is false
	assert.Nil(err)
}

func Test_ResolveVars(t *testing.T) {
	assert := assert.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	err := os.Setenv("MYVAR", ts.URL)
	assert.Nil(err)
	conf, err := config.LoadFromYAML([]byte(strings.TrimSpace(`
debugch: true
output:
  - type: elastic
    url: ["%{MYVAR}"]
    index: "gogstash-index-test"
    document_type: "testtype"
    document_id: "%{fieldstring}"
    bulk_actions: 0
	`)))
	assert.Nil(err)
	assert.NotNil(conf)
	resolvedConf, err := InitHandler(context.TODO(), &conf.OutputRaw[0])
	assert.Nil(err)
	outputConf := resolvedConf.(*OutputConfig)
	assert.Equal(ts.URL, outputConf.resolvedURLs[0])
}

func Test_output_elastic_module(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(assert)
	require := require.New(t)
	require.NotNil(require)

	ctx, cancel := context.WithTimeout(context.Background(), 2000*time.Millisecond)
	defer cancel()
	testIndexName := "gogstash-index-test"

	conf, err := config.LoadFromYAML([]byte(strings.TrimSpace(`
debugch: true
output:
  - type: elastic
    url: ["http://127.0.0.1:9200"]
    index: "gogstash-index-test"
    document_type: "testtype"
    document_id: "%{fieldstring}"
    bulk_actions: 0
	`)))
	require.NoError(err)
	err = conf.Start(ctx)
	if err != nil {
		require.True(ErrorCreateClientFailed1.In(err))
		t.Skip("skip test output elastic module")
	}

	client, err := elastic.NewClient(
		elastic.SetURL("http://127.0.0.1:9200"),
		elastic.SetSniff(false),
		elastic.SetDecoder(&jsonDecoder{}),
	)
	require.NoError(err)
	require.NotNil(client)

	defer func() {
		_, err = client.DeleteIndex(testIndexName).Do(ctx)
		require.NoError(err)
	}()

	conf.TestInputEvent(logevent.LogEvent{
		Timestamp: time.Date(2017, 4, 18, 19, 53, 1, 2, time.UTC),
		Message:   "output elastic test message",
		Extra: map[string]interface{}{
			"fieldstring": "ABC",
			"fieldnumber": 123,
		},
	})

	if event, err2 := conf.TestGetOutputEvent(300 * time.Millisecond); assert.NoError(err2) {
		require.Equal("output elastic test message", event.Message)
	}

	result, err := client.Get().Index(testIndexName).Id("ABC").Do(ctx)
	require.NoError(err)
	require.NotNil(result)
	require.NotNil(result.Source)
	require.JSONEq(`{"@timestamp":"2017-04-18T19:53:01.000000002Z","fieldnumber":123,"fieldstring":"ABC","message":"output elastic test message"}`, string(*result.Source))
}
