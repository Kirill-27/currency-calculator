package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/spf13/cast"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
  NBUUrl   = "https://bank.gov.ua/NBUStatService/v1/statdirectory/exchange?&json"
  RUBIndex = 18
  USDIndex = 26
  EURIndex = 32
)

type Result struct {
	USD      float64
	EUR      float64
	RUB      float64
}

type NBUResponse []struct {
	R030         int     `json:"r030"`
	Txt          string  `json:"txt"`
	Rate         float64 `json:"rate"`
	Cc           string  `json:"cc"`
	Exchangedate string  `json:"exchangedate"`
}

type ExchangeRatesKeeper struct {
	USD      float64
	EUR      float64
	RUB      float64
	Results  chan Result
}

func NewExchangeRatesKeeper() *ExchangeRatesKeeper {
	var response NBUResponse

	resp, err := http.Get(NBUUrl)
	if err != nil {
		log.Fatal("an error occurred when get from nbu api")
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Fatal("an error occurred when decode response from nbu api")
	}
	return &ExchangeRatesKeeper {
		USD: response[USDIndex].Rate,
		EUR: response[EURIndex].Rate,
		RUB: response[RUBIndex].Rate,
		Results: make(chan Result),
	}
}

func (e *ExchangeRatesKeeper) LastResultGetter() {
	for elem := range e.Results {
		fmt.Println(len(e.Results), elem.USD)
	}
}

func (e *ExchangeRatesKeeper) ExchangeRatesGetter() {
	for {
		time.Sleep(3 * time.Second)
		//time.Sleep(1 * time.Hour)
		var response NBUResponse

		resp, err := http.Get(NBUUrl)
		if err != nil {
			log.Fatal("an error occurred when get from nbu api")
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Fatal("an error occurred when decode response from nbu api")
		}
		e.USD = response[USDIndex].Rate
		e.EUR = response[EURIndex].Rate
		e.RUB = response[RUBIndex].Rate
		e.Results <- Result {
			USD: e.USD,
			EUR: e.EUR,
			RUB: e.RUB,
		}
	}
}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

func headers(w http.ResponseWriter, req *http.Request) {
	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func (e *ExchangeRatesKeeper) calculate(w http.ResponseWriter, req *http.Request) {
	price, _ := cast.ToIntE(chi.URLParam(req, "price"))
	currency, _ := cast.ToStringE(chi.URLParam(req, "currency"))

	switch strings.ToLower(currency) {
	case "usd":
		fmt.Fprintf(w, "%d USD = %f UAN", price, float64(price)*e.USD)
	case "eur":
		fmt.Fprintf(w, "%d EUR = %f UAN", price, float64(price)*e.EUR)
	case "rub":
		fmt.Fprintf(w, "%d RUB = %f UAN", price, float64(price)*e.RUB)
	default:
		fmt.Fprintf(w, "No such currency")
	}
}

func main() {

	exchangeRatesKeeper := NewExchangeRatesKeeper()
	go exchangeRatesKeeper.ExchangeRatesGetter()
	r := chi.NewRouter()
	r.Route("/calculate", func(r chi.Router) {
		r.Get("/{currency}/{price}", exchangeRatesKeeper.calculate)
	})
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/calculate/{currency}/{price}", exchangeRatesKeeper.calculate)
	http.HandleFunc("/headers", headers)
	http.ListenAndServe(":8384", r)

	//exchangeRatesKeeper.LastResultGetter()

}

