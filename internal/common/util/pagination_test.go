package util_test

import (
	"net/http/httptest"
	"testing"

	"github.com/philiplambok/tudu/internal/common/util"
)

func TestNewPagingRequestDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks", nil)
	paging := util.NewPagingRequest(req)
	if paging.Page != 1 {
		t.Fatalf("expected page 1, got %d", paging.Page)
	}
	if paging.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", paging.Limit)
	}
}

func TestNewPagingRequestValidParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks?page=2&limit=5", nil)
	paging := util.NewPagingRequest(req)
	if paging.Page != 2 {
		t.Fatalf("expected page 2, got %d", paging.Page)
	}
	if paging.Limit != 5 {
		t.Fatalf("expected limit 5, got %d", paging.Limit)
	}
}

func TestNewPagingRequestNormalizesInvalidParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks?page=abc&limit=-1", nil)
	paging := util.NewPagingRequest(req)
	if paging.Page != 1 {
		t.Fatalf("expected page 1, got %d", paging.Page)
	}
	if paging.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", paging.Limit)
	}
}

func TestNewPagingRequestCapsLimit(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks?page=1&limit=999", nil)
	paging := util.NewPagingRequest(req)
	if paging.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", paging.Limit)
	}
}

func TestPagingRequestOffset(t *testing.T) {
	paging := util.PagingRequest{Page: 2, Limit: 20}
	if paging.Offset() != 20 {
		t.Fatalf("expected offset 20, got %d", paging.Offset())
	}
}

func TestPagingResponsePageInfoMiddlePage(t *testing.T) {
	response := util.NewPagingResponse([]int{1, 2, 3, 4, 5}, int64(12), 2, 5)
	info := response.PageInfo()
	if info.Count != 5 {
		t.Fatalf("expected count 5, got %d", info.Count)
	}
	if info.CurrentPage != 2 {
		t.Fatalf("expected current page 2, got %d", info.CurrentPage)
	}
	if info.TotalData != 12 {
		t.Fatalf("expected total data 12, got %d", info.TotalData)
	}
	if info.TotalPage != 3 {
		t.Fatalf("expected total page 3, got %d", info.TotalPage)
	}
	if info.PreviousPage == nil || *info.PreviousPage != 1 {
		t.Fatalf("expected previous page 1, got %v", info.PreviousPage)
	}
	if info.NextPage == nil || *info.NextPage != 3 {
		t.Fatalf("expected next page 3, got %v", info.NextPage)
	}
}

func TestPagingResponsePageInfoFirstPage(t *testing.T) {
	response := util.NewPagingResponse([]int{1, 2, 3}, int64(8), 1, 3)
	info := response.PageInfo()
	if info.PreviousPage != nil {
		t.Fatalf("expected previous page nil, got %v", info.PreviousPage)
	}
	if info.NextPage == nil || *info.NextPage != 2 {
		t.Fatalf("expected next page 2, got %v", info.NextPage)
	}
}

func TestPagingResponsePageInfoLastPage(t *testing.T) {
	response := util.NewPagingResponse([]int{1, 2}, int64(8), 3, 3)
	info := response.PageInfo()
	if info.NextPage != nil {
		t.Fatalf("expected next page nil, got %v", info.NextPage)
	}
	if info.PreviousPage == nil || *info.PreviousPage != 2 {
		t.Fatalf("expected previous page 2, got %v", info.PreviousPage)
	}
}

func TestPagingResponsePageInfoZeroTotal(t *testing.T) {
	response := util.NewPagingResponse([]int{}, int64(0), 1, 20)
	info := response.PageInfo()
	if info.TotalPage != 1 {
		t.Fatalf("expected total page 1, got %d", info.TotalPage)
	}
	if info.NextPage != nil {
		t.Fatalf("expected next page nil, got %v", info.NextPage)
	}
	if info.PreviousPage != nil {
		t.Fatalf("expected previous page nil, got %v", info.PreviousPage)
	}
}
