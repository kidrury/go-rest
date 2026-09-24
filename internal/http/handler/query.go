package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/kidrury/rest-pro/internal/domain"
)

type Query struct {
	values url.Values
}

func ParseQuery(r *http.Request) (Query, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return Query{}, domain.Error{
			Code:    "INVALID_REQUEST",
			Message: "query string contains invalid encoding",
		}
	}
	return Query{values: values}, nil
}

func (q Query) RejectUnknown(allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))

	for _, entry := range allowed {
		allowedSet[entry] = struct{}{}
	}

	for val := range q.values {
		if _, ok := allowedSet[val]; !ok {
			return domain.Error{
				Code:    "INVALID_QUERY_PARAMETER",
				Message: fmt.Sprintf("query parameter %q is not supported", val),
			}
		}
	}
	return nil
}

func (q Query) Int64(name string, defaultValue int64) (int64, error) {
	values := q.values[name]

	if len(values) == 0 {
		return defaultValue, nil
	}

	if len(values) > 1 {
		return 0, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must appear only once", name),
		}
	}

	if values[0] == "" {
		return 0, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must have a value", name),
		}
	}

	parsed, err := strconv.ParseInt(values[0], 10, 64)

	if err != nil {
		return 0, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must be a valid integer", name),
		}
	}

	return parsed, nil
}

func (q Query) String(name string) (string, error) {
	values := q.values[name]

	if len(values) == 0 {
		return "", domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q is required", name),
		}
	}

	if len(values) > 1 {
		return "", domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must appear only once", name),
		}
	}

	if values[0] == "" {
		return "", domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must have a value", name),
		}
	}

	return values[0], nil
}

func (q Query) Bool(name string) (bool, error) {
	values := q.values[name]

	if len(values) == 0 {
		return false, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q is required", name),
		}
	}

	if len(values) > 1 {
		return false, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must appear only once", name),
		}
	}

	if values[0] == "" {
		return false, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must have a value", name),
		}
	}

	parsed, err := strconv.ParseBool(values[0])

	if err != nil {
		return false, domain.Error{
			Code:    "INVALID_QUERY_PARAMETER",
			Message: fmt.Sprintf("query parameter %q must be a valid boolean", name),
		}
	}

	return parsed, nil
}
