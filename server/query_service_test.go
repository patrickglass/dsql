package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery_SQL(t *testing.T) {
	sql := `
		with h as (
			select  min("bookings"), (select a from ss) as d from "data-model"
		)
		select * from h
		`
	expected := QueryResponse{
		Headers: []string{
			"account-region-hq",
			"avg-exp-deal-size",
			"avg-new-deal-size",
		},
		Rows: [][]interface{}{
			{
				"AMER",
				nil,
				13578.211943824490,
			},
			{
				"APAC",
				123.0,
				13976.055459466232,
			},
		},
	}

	data := FetchMetrics(sql, "localhost:7777", "c3c1e112-4353-4cd5-90f4-2ada9258bacc")
	assert.Equal(t, expected, data)
}
