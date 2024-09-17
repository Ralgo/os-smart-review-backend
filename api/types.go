package api

type ProductResponse struct {
	ExternalID      string           `json:"id"`
	AverageRating   float32          `json:"average_rating"`
	ReviewsQuantity int              `json:"reviews_quantity"`
	AISummary       string           `json:"ai_summary,omitempty"`
	Keywords        []string         `json:"keywords,omitempty"`
	Reviews         []ReviewResponse `json:"reviews"`
}

type ProductListResponse struct {
	Products []ProductResponse `json:"products"`
}

type ReviewResponse struct {
	Author     string `json:"author"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Rating     int    `json:"rating"`
	ReviewDate string `json:"review_date"`
}

type ProductInput struct {
	ShopID string      `json:"shop_id" binding:"required"`
	Review ReviewInput `json:"review" binding:"required"`
}

type ReviewInput struct {
	Author  string `json:"author" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Rating  int    `json:"rating" binding:"required"`
}
