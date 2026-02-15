package port

// ListMeta is returned with paginated list responses.
type ListMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
}

// ListResponse wraps items and meta for list endpoints.
type ListResponse struct {
	Items interface{} `json:"items"`
	Meta  ListMeta    `json:"meta"`
}
