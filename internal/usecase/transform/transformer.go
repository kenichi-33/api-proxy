package transform

import (
	"net/http"

	"api-proxy/internal/domain"
)

type TransformStrategy interface {
	TransformRequest(r *http.Request, config map[string]interface{}) error
	TransformResponse(res *http.Response, config map[string]interface{}) error
}

type Transformer interface {
	TransformRequest(r *http.Request, transforms []domain.Transformation) error
	TransformResponse(res *http.Response, transforms []domain.Transformation) error
}

type transformer struct {
	strategies map[string]TransformStrategy
}

func NewTransformer() Transformer {
	return &transformer{
		strategies: map[string]TransformStrategy{
			"header": &HeaderTransformStrategy{},
			"casis":  &CasisTransformStrategy{},
		},
	}
}

func (t *transformer) TransformRequest(r *http.Request, transforms []domain.Transformation) error {
	for _, tr := range transforms {
		if strategy, ok := t.strategies[tr.Type]; ok {
			strategy.TransformRequest(r, tr.Req)
		}
	}
	return nil
}

func (t *transformer) TransformResponse(res *http.Response, transforms []domain.Transformation) error {
	for _, tr := range transforms {
		if strategy, ok := t.strategies[tr.Type]; ok {
			strategy.TransformResponse(res, tr.Res)
		}
	}
	return nil
}

// Strategies

type HeaderTransformStrategy struct{}

func (s *HeaderTransformStrategy) TransformRequest(r *http.Request, config map[string]interface{}) error {
	if config == nil {
		return nil
	}

	// Add Headers
	if val, ok := config["add_headers"]; ok {
		if addHeaders, ok := val.(map[string]interface{}); ok {
			for k, v := range addHeaders {
				if s, ok := v.(string); ok {
					r.Header.Set(k, s)
				}
			}
		}
	}
	// Remove Headers
	if val, ok := config["remove_headers"]; ok {
		if removeHeaders, ok := val.([]interface{}); ok {
			for _, v := range removeHeaders {
				if s, ok := v.(string); ok {
					r.Header.Del(s)
				}
			}
		}
	}
	return nil
}

func (s *HeaderTransformStrategy) TransformResponse(res *http.Response, config map[string]interface{}) error {
	if config == nil {
		return nil
	}

	// Add Headers
	if val, ok := config["add_headers"]; ok {
		if addHeaders, ok := val.(map[string]interface{}); ok {
			for k, v := range addHeaders {
				if s, ok := v.(string); ok {
					res.Header.Set(k, s)
				}
			}
		}
	}
	// Remove Headers
	if val, ok := config["remove_headers"]; ok {
		if removeHeaders, ok := val.([]interface{}); ok {
			for _, v := range removeHeaders {
				if s, ok := v.(string); ok {
					res.Header.Del(s)
				}
			}
		}
	}
	return nil
}

type CasisTransformStrategy struct{}

func (s *CasisTransformStrategy) TransformRequest(r *http.Request, config map[string]interface{}) error {
	// Placeholder
	return nil
}

func (s *CasisTransformStrategy) TransformResponse(res *http.Response, config map[string]interface{}) error {
	// Placeholder
	return nil
}
