package util

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type PagingRequest struct {
	Page  int
	Limit int
}

func NewPagingRequest(r *http.Request) PagingRequest {
	page := parsePositiveInt(r.URL.Query().Get("page"), DefaultPage)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), DefaultLimit)
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return PagingRequest{Page: page, Limit: limit}
}

func (p PagingRequest) Offset() int {
	return (p.Page - 1) * p.Limit
}

type PageInfo struct {
	Count        int  `json:"count"`
	CurrentPage  int  `json:"current_page"`
	NextPage     *int `json:"next_page,omitempty"`
	PreviousPage *int `json:"previous_page,omitempty"`
	TotalData    int  `json:"total_data"`
	TotalPage    int  `json:"total_page"`
}

type PagingResponse[T any] struct {
	data      []T
	totalData int64
	page      int
	limit     int
}

func NewPagingResponse[T any](data []T, total int64, page int, limit int) PagingResponse[T] {
	return PagingResponse[T]{
		data:      data,
		totalData: total,
		page:      page,
		limit:     limit,
	}
}

func (p PagingResponse[T]) Data() []T {
	return p.data
}

func (p PagingResponse[T]) PageInfo() PageInfo {
	limit := p.limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	page := p.page
	if page <= 0 {
		page = DefaultPage
	}

	totalPage := 1
	if p.totalData > 0 {
		totalPage = int((p.totalData + int64(limit) - 1) / int64(limit))
	}

	var previousPage *int
	if page > 1 {
		previousPage = intPtr(page - 1)
	}

	var nextPage *int
	if page < totalPage {
		nextPage = intPtr(page + 1)
	}

	return PageInfo{
		Count:        len(p.data),
		CurrentPage:  page,
		NextPage:     nextPage,
		PreviousPage: previousPage,
		TotalData:    int(p.totalData),
		TotalPage:    totalPage,
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intPtr(v int) *int {
	return &v
}
