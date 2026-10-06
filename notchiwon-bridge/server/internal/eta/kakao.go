package eta

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultKakaoURL is the Kakao Mobility car directions endpoint.
const DefaultKakaoURL = "https://apis-navi.kakaomobility.com/v1/directions"

// Kakao estimates the ETA with Kakao Mobility car directions (real-time
// traffic). When the elder's home is not registered, or the API fails, it
// answers with Fallback so a location update never fails because of the map
// provider: the caregiver keeps driving and the next update tries again.
type Kakao struct {
	// APIKey is the Kakao REST API key (`Authorization: KakaoAK ...`).
	APIKey string
	// URL defaults to DefaultKakaoURL.
	URL string
	// HTTP defaults to a client with a 5 second timeout.
	HTTP *http.Client
	// Fallback answers when Kakao cannot. Defaults to Schedule{}.
	Fallback Estimator
	Logger   *slog.Logger
}

// Minutes implements Estimator.
func (k Kakao) Minutes(ctx context.Context, r Request) (int, error) {
	fallback := k.Fallback
	if fallback == nil {
		fallback = Schedule{}
	}
	if r.Destination == nil {
		return fallback.Minutes(ctx, r)
	}
	d, err := k.duration(ctx, Point{r.Latitude, r.Longitude}, *r.Destination)
	if err != nil {
		if k.Logger != nil {
			k.Logger.WarnContext(ctx, "kakao directions failed, using fallback eta", "err", err)
		}
		return fallback.Minutes(ctx, r)
	}
	return ceilMinutes(d), nil
}

// directions is the part of the Kakao response we read.
type directions struct {
	Routes []struct {
		ResultCode int    `json:"result_code"`
		ResultMsg  string `json:"result_msg"`
		Summary    *struct {
			Duration int `json:"duration"` // seconds
		} `json:"summary"`
	} `json:"routes"`
}

// resultTooClose is Kakao's result_code when origin and destination are
// within 5 m of each other: the caregiver is there.
const resultTooClose = 104

func (k Kakao) duration(ctx context.Context, from, to Point) (time.Duration, error) {
	base := k.URL
	if base == "" {
		base = DefaultKakaoURL
	}
	q := url.Values{
		// Kakao takes "longitude,latitude".
		"origin":      {coord(from)},
		"destination": {coord(to)},
		"priority":    {"RECOMMEND"},
		"summary":     {"true"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+q.Encode(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "KakaoAK "+k.APIKey)

	client := k.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("kakao directions: HTTP %d", resp.StatusCode)
	}
	var body directions
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode kakao directions: %w", err)
	}
	if len(body.Routes) == 0 {
		return 0, fmt.Errorf("kakao directions: no route")
	}
	route := body.Routes[0]
	switch {
	case route.ResultCode == resultTooClose:
		return 0, nil
	case route.ResultCode != 0 || route.Summary == nil:
		return 0, fmt.Errorf("kakao directions: %d %s", route.ResultCode, route.ResultMsg)
	}
	return time.Duration(route.Summary.Duration) * time.Second, nil
}

func coord(p Point) string {
	return strconv.FormatFloat(p.Longitude, 'f', 6, 64) + "," + strconv.FormatFloat(p.Latitude, 'f', 6, 64)
}
