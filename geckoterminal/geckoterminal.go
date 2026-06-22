package geckoterminal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	httpClient *http.Client
	limiter    *rate.Limiter
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Every(2*time.Second), 3),
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.limiter.Wait(context.Background())
	return c.httpClient.Do(req)
}

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return e.Message
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

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{Status: resp.StatusCode, Message: fmt.Sprintf("geckoterminal: status %d: %s", resp.StatusCode, string(body))}
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
	BaseImage    string  `json:"baseImage"`
	PriceUSD     string  `json:"priceUsd"`
	FDV          float64 `json:"fdv"`
	MarketCap    float64 `json:"marketCap"`
	TotalSupply  float64 `json:"totalSupply"`
	PriceChange24h string `json:"priceChange24h"`
	Volume24h    string  `json:"volume24h"`
	Liquidity    string  `json:"liquidity"`
}

type poolResponse struct {
	Data struct {
		Attributes struct {
			Name               string `json:"name"`
			BaseTokenPriceUSD  string `json:"base_token_price_usd"`
			FDVInUSD           string `json:"fdv_usd"`
			MarketCapUSD       string `json:"market_cap_usd"`
			PriceChangeH24     string `json:"price_change_percentage"`
			VolumeH24          struct {
				H24 string `json:"h24"`
			} `json:"volume_usd"`
			ReserveUSD         string `json:"reserve_in_usd"`
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
			Name     string `json:"name"`
			Symbol   string `json:"symbol"`
			ImageURL string `json:"image_url"`
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

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{Status: resp.StatusCode, Message: fmt.Sprintf("geckoterminal: status %d: %s", resp.StatusCode, string(body))}
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
		Name:           result.Data.Attributes.Name,
		PriceUSD:       priceUSD,
		FDV:            fdv,
		MarketCap:      mcap,
		TotalSupply:    totalSupply,
		PriceChange24h: result.Data.Attributes.PriceChangeH24,
		Volume24h:      result.Data.Attributes.VolumeH24.H24,
		Liquidity:      result.Data.Attributes.ReserveUSD,
	}

	tokenID := result.Data.Relationships.BaseToken.Data.ID
	if tokenID != "" {
		tokenInfo, err := c.getTokenInfo(tokenID)
		if err == nil {
			details.BaseSymbol = tokenInfo.symbol
			details.BaseImage = tokenInfo.imageURL
			if tokenInfo.name != "" {
				details.Name = tokenInfo.name
			}
		}
	}

	return details, nil
}

type tokenInfo struct {
	name     string
	symbol   string
	imageURL string
}

func (c *Client) getTokenInfo(tokenID string) (*tokenInfo, error) {
	url := fmt.Sprintf("https://api.geckoterminal.com/api/v2/networks/%s", tokenID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token info: status %d", resp.StatusCode)
	}

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &tokenInfo{
		name:     result.Data.Attributes.Name,
		symbol:   result.Data.Attributes.Symbol,
		imageURL: result.Data.Attributes.ImageURL,
	}, nil
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

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{Status: resp.StatusCode, Message: fmt.Sprintf("geckoterminal: status %d: %s", resp.StatusCode, string(body))}
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
