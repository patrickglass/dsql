package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/api/v1/executeSQL", func(w http.ResponseWriter, r *http.Request) {
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Println(name, value)
			}
		}

		w.Write([]byte(`{
	    "headers": [
	        "account-region-hq",
	        "avg-exp-deal-size",
	        "avg-new-deal-size"
	    ],
	    "rows": [
	        [
	            "AMER",
	            null,
	            13578.211943824490
	        ],
	        [
	            "APAC",
	            123,
	            13976.055459466231
	        ]
	    ]
	}`))
	})

	log.Fatal(http.ListenAndServe(":7777", nil))
}
