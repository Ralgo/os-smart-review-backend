package api

import (
	"net/http"
	"os-smart-reviews-backend/store"
	"strings"

	"github.com/gin-gonic/gin"
)

type API struct {
	engine *gin.Engine
}

func setupRouter(s store.Store) *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Get product reviews
	r.GET("/product/:external_id/review", func(c *gin.Context) {
		externalID := c.Params.ByName("external_id")
		product, err := s.GetProductAndReviewsByExternalID(externalID)
		if product == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if product.Reviews == nil || len(product.Reviews) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reviews not found"})
		}

		response := parseResponse(product)
		averageRating, err := s.GetAverageRatingByProductExternalID(externalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// parse float64 to string
		response.AverageRating = averageRating

		reviewsQuantity, err := s.GetTotalReviewsQuantityByProductExternalID(externalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response.ReviewsQuantity = reviewsQuantity
		c.JSON(http.StatusOK, response)
	})

	// Get all products by a given shop ID
	r.GET("/shop/:shop_id/product", func(c *gin.Context) {
		shopID := c.Params.ByName("shop_id")
		products, err := s.GetProductsByShopID(shopID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if products == nil || len(products) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Products not found"})
			return
		}

		response := parseProductsResponse(s, products)
		c.JSON(http.StatusOK, response)
	})

	// Create review and if the product does not exist, create it too
	r.POST("/product/:external_id/review", func(c *gin.Context) {
		externalID := c.Params.ByName("external_id")
		var productReview ProductInput
		if err := c.BindJSON(&productReview); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		product, err := s.GetProductByExternalID(externalID)
		if err != nil {
			if err.Error() == "product not found: record not found" {
				product = &store.Product{
					ExternalID: externalID,
					ShopID:     productReview.ShopID,
				}
				if _, err := s.CreateProduct(product); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		err = s.CreateSmartReview(product.ExternalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		err = s.CreateReviewsKeywords(product.ExternalID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result := parseReviewInput(product.ID, productReview.Review)
		if _, err := s.CreateReview(&result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Review created"})
	})

	return r
}

func New(store store.Store) (*API, error) {
	return &API{
		engine: setupRouter(store),
	}, nil
}

func (a *API) Run(port string) {
	a.engine.Run(":" + port)
}

func parseResponse(product *store.Product) ProductResponse {
	var reviews []ReviewResponse
	var aiSummary string
	for _, review := range product.Reviews {
		if !review.IAGenerated {
			reviews = append(reviews, ReviewResponse{
				Author:     review.Author,
				Title:      review.Title,
				Content:    review.Content,
				Rating:     review.Rating,
				ReviewDate: review.CreatedAt.String(),
			})
		} else {
			aiSummary = review.Content
		}
	}
	response := ProductResponse{
		ExternalID: product.ExternalID,
		Reviews:    reviews,
		Keywords:   formatKeywords(product.Keywords),
		AISummary:  aiSummary,
	}

	return response
}

func parseReviewInput(id uint, review ReviewInput) store.Review {
	return store.Review{
		ProductID: id,
		Author:    review.Author,
		Title:     review.Title,
		Content:   review.Content,
		Rating:    review.Rating,
	}
}

func parseProductsResponse(s store.Store, products []store.Product) []ProductResponse {
	var response []ProductResponse
	for _, product := range products {

		averageRating, err := s.GetAverageRatingByProductExternalID(product.ExternalID)
		if err != nil {
			continue
		}

		totalReviewsQuantity, err := s.GetTotalReviewsQuantityByProductExternalID(product.ExternalID)
		if err != nil {
			continue
		}

		response = append(response, ProductResponse{
			ExternalID:      product.ExternalID,
			AverageRating:   averageRating,
			ReviewsQuantity: totalReviewsQuantity,
			Reviews:         nil,
			AISummary:       "",
		})
	}
	return response
}

func formatKeywords(keywords string) []string {
	return strings.Split(keywords, ",")
}
