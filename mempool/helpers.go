package mempool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	retryDelay                  = 60 * time.Second
	maxRetryAttempts            = 3
	chartViewOption             = "chart"
	defaultViewOption           = chartViewOption
	mempoolDefaultChartDataType = "size"
	maxPageSize                 = 250
	defaultPageSize             = 20
	noDataMessage               = "does not have data for the selected query option(s)."
)

var (
	pageSizeSelector = map[int]int{
		20:  20,
		30:  30,
		50:  50,
		100: 100,
		150: 150,
	}
)

// GetResponse attempts to collect json data from the given url string and decodes it into
// the destination
func GetResponse(ctx context.Context, client *http.Client, url string, destination interface{}) error {
	// if client has no timeout, set one
	if client.Timeout == time.Duration(0) {
		client.Timeout = 10 * time.Second
	}
	resp := new(http.Response)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	for i := 1; i <= maxRetryAttempts; i++ {
		res, err := client.Do(req)
		if err != nil {
			if res != nil {
				res.Body.Close()
			}
			if i == maxRetryAttempts {
				return err
			}
			time.Sleep(retryDelay)
			continue
		}
		resp = res
		break
	}

	err = json.NewDecoder(resp.Body).Decode(destination)
	if err != nil {
		return err
	}
	return nil
}

func AddParams(base string, params map[string]interface{}) (string, error) {
	var strBuilder strings.Builder

	_, err := strBuilder.WriteString(base)

	if err != nil {
		return base, err
	}

	strBuilder.WriteString("?")

	for param, value := range params {
		strBuilder.WriteString(param)
		strBuilder.WriteString("=")

		vType := reflect.TypeOf(value)
		switch vType.Kind() {
		case reflect.String:
			strBuilder.WriteString(reflect.ValueOf(value).String())
		case reflect.Int64, reflect.Int:
			strBuilder.WriteString(strconv.FormatInt(reflect.ValueOf(value).Int(), 10))
		case reflect.Float64:
			strBuilder.WriteString(strconv.FormatFloat(reflect.ValueOf(value).Float(), 'f', -1, 64))
		default:
			return strBuilder.String(), fmt.Errorf("unsupported type: %v", vType.Kind())
		}

		strBuilder.WriteString("&")
	}

	str := strBuilder.String()
	return str[:len(str)-1], nil
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func UnixTime(t int64) time.Time {
	return time.Unix(t, 0).UTC()
}

func DurationToString(duration time.Duration) string {
	duration = duration.Round(10 * time.Millisecond)
	return duration.String()
}
