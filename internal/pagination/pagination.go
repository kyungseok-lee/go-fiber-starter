package pagination

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// PageQuery는 offset 기반 페이지네이션 공통 파라미터다.
// 대량 테이블(수백만 건)이 되면 cursor 기반으로 전환할 것 (README Roadmap 참고).
type PageQuery struct {
	Page  int
	Limit int
}

func (p PageQuery) Offset() int { return (p.Page - 1) * p.Limit }

func NewPageQuery(page, limit int) PageQuery {
	if page < DefaultPage {
		page = DefaultPage
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit // 조회 행 수와 응답 크기 제한
	}
	return PageQuery{Page: page, Limit: limit}
}

// PageMeta는 목록 응답의 meta 블록이다.
type PageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalCount int64 `json:"total_count"`
	TotalPages int64 `json:"total_pages"`
}

func NewPageMeta(q PageQuery, total int64) PageMeta {
	tp := (total + int64(q.Limit) - 1) / int64(q.Limit)
	return PageMeta{Page: q.Page, Limit: q.Limit, TotalCount: total, TotalPages: tp}
}
