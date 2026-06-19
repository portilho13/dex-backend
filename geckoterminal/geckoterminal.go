package geckoterminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Candle struct {
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    float64 `json:"v"`
	Timestamp int64   `json:"unixTime"`
}

type ohlcvResponse struct {
	Data struct {
		Attributes struct {
			OHLCVList [][]float64 `json:"ohlcv_list"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetOHLCV(poolAddress string, aggregate string, timeframe string) ([]Candle, error) {
	url := fmt.Sprintf(
		"https://api.geckoterminal.com/api/v2/networks/solana/pools/%s/ohlcv/%s?aggregate=%s&limit=1000",
		poolAddress, timeframe, aggregate,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("geckoterminal: status %d: %s", resp.StatusCode, string(body))
	}

	var result ohlcvResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("geckoterminal: decode error: %w", err)
	}

	candles := make([]Candle, len(result.Data.Attributes.OHLCVList))
	for i, bar := range result.Data.Attributes.OHLCVList {
		if len(bar) < 6 {
			continue
		}
		candles[i] = Candle{
			Timestamp: int64(bar[0]),
			Open:      bar[1],
			High:      bar[2],
			Low:       bar[3],
			Close:     bar[4],
			Volume:    bar[5],
		}
	}

	return candles, nil
}

type PoolDetails struct {
	Name         string  `json:"name"`
	BaseSymbol   string  `json:"baseSymbol"`
	QuoteSymbol  string  `json:"quoteSymbol"`
	PriceUSD     string  `json:"priceUsd"`
	FDV          float64 `json:"fdv"`
	MarketCap    float64 `json:"marketCap"`
	TotalSupply  float64 `json:"totalSupply"`
}

type poolResponse struct {
	Data struct {
		Attributes struct {
			Name               string `json:"name"`
			BaseTokenPriceUSD  string `json:"base_token_price_usd"`
			FDVInUSD           string `json:"fdv_usd"`
			MarketCapUSD       string `json:"market_cap_usd"`
		} `json:"attributes"`
		Relationships struct {
			BaseToken struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"base_token"`
		} `json:"relationships"`
	} `json:"data"`
}

type tokenResponse struct {
	Data struct {
		Attributes struct {
			Symbol      string `json:"symbol"`
			TotalSupply string `json:"total_supply_in_token"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetPoolDetails(poolAddress string) (*PoolDetails, error) {
	url := fmt.Sprintf(
		"https://api.geckoterminal.com/api/v2/networks/solana/pools/%s",
		poolAddress,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("geckoterminal: status %d: %s", resp.StatusCode, string(body))
	}

	var result poolResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("geckoterminal: decode error: %w", err)
	}

	priceUSD := result.Data.Attributes.BaseTokenPriceUSD

	var fdv, mcap float64
	fmt.Sscanf(result.Data.Attributes.FDVInUSD, "%f", &fdv)
	fmt.Sscanf(result.Data.Attributes.MarketCapUSD, "%f", &mcap)

	var totalSupply float64
	if priceUSD != "" && priceUSD != "0" {
		var p float64
		fmt.Sscanf(priceUSD, "%f", &p)
		if p > 0 && fdv > 0 {
			totalSupply = fdv / p
		}
	}

	details := &PoolDetails{
		Name:        result.Data.Attributes.Name,
		PriceUSD:    priceUSD,
		FDV:         fdv,
		MarketCap:   mcap,
		TotalSupply: totalSupply,
	}

	return details, nil
}

type Trade struct {
	Timestamp      string  `json:"timestamp"`
	Kind           string  `json:"kind"`
	VolumeUSD      float64 `json:"volumeUsd"`
	VolumeBase     float64 `json:"volumeBase"`
	VolumeQuote    float64 `json:"volumeQuote"`
	PriceUSD       float64 `json:"priceUsd"`
	TxHash         string  `json:"txHash"`
	Maker          string  `json:"maker"`
}

type tradesResponse struct {
	Data []struct {
		Attributes struct {
			BlockTimestamp  string `json:"block_timestamp"`
			Kind           string `json:"kind"`
			VolumeInUSD    string `json:"volume_in_usd"`
			FromTokenAmount string `json:"from_token_amount"`
			ToTokenAmount   string `json:"to_token_amount"`
			PriceFromInUSD string `json:"price_from_in_usd"`
			PriceToInUSD   string `json:"price_to_in_usd"`
			TxHashURL      string `json:"tx_hash_url"`
			TxHash         string `json:"tx_hash"`
			TxFromAddress  string `json:"tx_from_address"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetTrades(poolAddress string) ([]Trade, error) {
	url := fmt.Sprintf(
		"https://api.geckoterminal.com/api/v2/networks/solana/pools/%s/trades",
		poolAddress,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("geckoterminal: status %d: %s", resp.StatusCode, string(body))
	}

	var result tradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("geckoterminal: decode error: %w", err)
	}

	trades := make([]Trade, 0, len(result.Data))
	for _, d := range result.Data {
		var volUSD, volBase, volQuote, priceUSD float64
		fmt.Sscanf(d.Attributes.VolumeInUSD, "%f", &volUSD)
		fmt.Sscanf(d.Attributes.FromTokenAmount, "%f", &volBase)
		fmt.Sscanf(d.Attributes.ToTokenAmount, "%f", &volQuote)
		fmt.Sscanf(d.Attributes.PriceToInUSD, "%f", &priceUSD)

		if d.Attributes.Kind == "buy" {
			fmt.Sscanf(d.Attributes.PriceFromInUSD, "%f", &priceUSD)
			volBase = volQuote
			fmt.Sscanf(d.Attributes.ToTokenAmount, "%f", &volQuote)
			fmt.Sscanf(d.Attributes.FromTokenAmount, "%f", &volBase)
		}

		txHash := d.Attributes.TxHash
		if txHash == "" {
			txHash = d.Attributes.TxHashURL
		}

		trades = append(trades, Trade{
			Timestamp:   d.Attributes.BlockTimestamp,
			Kind:        d.Attributes.Kind,
			VolumeUSD:   volUSD,
			VolumeBase:  volBase,
			VolumeQuote: volQuote,
			PriceUSD:    priceUSD,
			TxHash:      txHash,
			Maker:       d.Attributes.TxFromAddress,
		})
	}

	return trades, nil
}
