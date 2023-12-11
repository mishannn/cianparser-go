package httpclient

import "net/http"

type RoundTripperWithHeaders struct {
	r      http.RoundTripper
	header http.Header
}

func NewRoundTripperWithHeaders(r http.RoundTripper, header http.Header) *RoundTripperWithHeaders {
	return &RoundTripperWithHeaders{
		r:      r,
		header: header,
	}
}

func (rt *RoundTripperWithHeaders) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, vs := range rt.header {
		for _, v := range vs {
			r.Header.Set(k, v)
		}
	}

	return rt.r.RoundTrip(r)
}
