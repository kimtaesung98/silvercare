package eta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func kakaoServer(t *testing.T, status int, body string, seen *http.Request) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = *r
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fixed is a fallback that always answers 42, so tests can tell it was used.
type fixed struct{}

func (fixed) Minutes(context.Context, Request) (int, error) { return 42, nil }

var home = &Point{Latitude: 37.5665, Longitude: 126.9780}

func TestKakaoMinutes(t *testing.T) {
	var seen http.Request
	srv := kakaoServer(t, 200, `{"routes":[{"result_code":0,"result_msg":"길 찾기 성공","summary":{"distance":5200,"duration":610}}]}`, &seen)
	k := Kakao{APIKey: "key", URL: srv.URL, Fallback: fixed{}}

	got, err := k.Minutes(context.Background(), Request{Latitude: 37.55, Longitude: 126.97, Destination: home})
	if err != nil || got != 11 {
		t.Fatalf("got %d, %v; want 11 (610s rounded up)", got, err)
	}
	if h := seen.Header.Get("Authorization"); h != "KakaoAK key" {
		t.Errorf("Authorization = %q", h)
	}
	q := seen.URL.Query()
	if q.Get("origin") != "126.970000,37.550000" || q.Get("destination") != "126.978000,37.566500" {
		t.Errorf("query = %v (Kakao takes longitude first)", q)
	}
}

func TestKakaoArrived(t *testing.T) {
	srv := kakaoServer(t, 200, `{"routes":[{"result_code":104,"result_msg":"출발지와 도착지가 5 m 이내로 설정된 경우 경로를 탐색할 수 없음"}]}`, nil)
	got, err := Kakao{APIKey: "key", URL: srv.URL, Fallback: fixed{}}.Minutes(context.Background(), Request{Destination: home})
	if err != nil || got != 0 {
		t.Fatalf("got %d, %v; want 0", got, err)
	}
}

func TestKakaoFallsBack(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		dest   *Point
	}{
		"no home registered": {200, `{}`, nil},
		"http error":         {401, `{"msg":"unauthorized"}`, home},
		"no route":           {200, `{"routes":[{"result_code":1,"result_msg":"경로 없음"}]}`, home},
		"empty routes":       {200, `{"routes":[]}`, home},
		"broken json":        {200, `{`, home},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			srv := kakaoServer(t, c.status, c.body, nil)
			got, err := Kakao{APIKey: "key", URL: srv.URL, Fallback: fixed{}}.Minutes(context.Background(), Request{Destination: c.dest})
			if err != nil || got != 42 {
				t.Fatalf("got %d, %v; want the fallback's 42", got, err)
			}
		})
	}
}

func TestKakaoTimeoutFallsBack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	k := Kakao{APIKey: "key", URL: srv.URL, Fallback: fixed{}, HTTP: &http.Client{Timeout: 50 * time.Millisecond}}
	got, err := k.Minutes(context.Background(), Request{Destination: home})
	if err != nil || got != 42 {
		t.Fatalf("got %d, %v; want the fallback's 42", got, err)
	}
}
