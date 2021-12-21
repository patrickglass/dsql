package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type QueryResponse struct {
	Headers []string        `json:"headers"`
	Rows    [][]interface{} `json:"rows"`
}

func FetchMetrics(sql, host, tenantID string) QueryResponse {
	data := QueryResponse{}
	sqlStmt := strings.NewReader(sql)

	url := fmt.Sprintf("http://%s/api/v1/executeSQL", host)

	req, err := http.NewRequest("POST", url, sqlStmt)
	if err != nil {
		log.Error().Err(err).Msg("could not build request")
		return data
	}
	req.Header.Set("X-TenantId", tenantID)
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("request to query server failed")
		return data
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("could not read response body")
		return data
	}

	fmt.Println(string(body))
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse json response")
		return data
	}

	return data
}
