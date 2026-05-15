package repositories

type PaginationMeta struct {
	Total       int64
	CurrentPage int
	TotalPages  int
	Limit       int
}

type PaginatedResult[T any] struct {
	Items []T
	Meta  PaginationMeta
}

func NewPaginationMeta(total int64, page, limit int) PaginationMeta {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	if totalPages == 0 && total > 0 {
		totalPages = 1
	}

	return PaginationMeta{
		Total:       total,
		CurrentPage: page,
		TotalPages:  totalPages,
		Limit:       limit,
	}
}
