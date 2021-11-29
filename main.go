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
	UAN           int
	Currency      string
	ResultValue   float64
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
func (r Result) ToString() string{
	return fmt.Sprintf("UAN: %d, Currency: %s, ResultValue: %f", r.UAN, r.Currency, r.ResultValue)
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

func (e *ExchangeRatesKeeper) ExchangeRatesGetter() {
	for {
		time.Sleep(3 * time.Second)
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
	}
}

func (e *ExchangeRatesKeeper) CalculatePrise(w http.ResponseWriter, req *http.Request) {
	price, _ := cast.ToIntE(chi.URLParam(req, "price"))
	currency, _ := cast.ToStringE(chi.URLParam(req, "currency"))

	var result Result
	result.UAN = price
	result.Currency = strings.ToLower(currency)
	switch strings.ToLower(currency) {
	case "usd":
		summa :=  float64(price)*e.USD
		fmt.Fprintf(w, "%d USD = %f UAN", price, float64(price)*e.USD)
		result.ResultValue = summa
	case "eur":
		summa :=  float64(price)*e.EUR
		fmt.Fprintf(w, "%d EUR = %f UAN", price, summa)
		result.ResultValue = summa
	case "rub":
		summa :=  float64(price)*e.RUB
		fmt.Fprintf(w, "%d RUB = %f UAN", price, summa)
		result.ResultValue = summa
	default:
		fmt.Fprintf(w, "No such currency")
		return
	}

	e.Results <- result
}

func (e *ExchangeRatesKeeper) GetLastResult(w http.ResponseWriter, req *http.Request) {
	if len(e.Results) == 0 {
		fmt.Fprintf(w, "No raw results in history")
		return
	}
	res := <- e.Results
	fmt.Fprintf(w, "Last unreaded result:\n"+res.ToString())
}

func main() {
	exchangeRatesKeeper := NewExchangeRatesKeeper()
	go exchangeRatesKeeper.ExchangeRatesGetter()
	r := chi.NewRouter()
	r.Route("/calculate", func(r chi.Router) {
		r.Get("/{currency}/{price}", exchangeRatesKeeper.CalculatePrise)
	})
	go r.Route("/lastresult", func(r chi.Router) {
		r.Get("/", exchangeRatesKeeper.GetLastResult)
	})
	http.ListenAndServe(":8384", r)
}
