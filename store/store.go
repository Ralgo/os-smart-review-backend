package store

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henomis/lingoose/llm/antropic"
	"github.com/henomis/lingoose/thread"
)

type Store struct {
	ctx      context.Context
	database *gorm.DB
	ai       *antropic.Antropic
}

func New(ctx context.Context, dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Migrate the schemas
	if err := db.AutoMigrate(&Product{}, &Review{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	// Register the models for the antropic package
	antropicllm := antropic.New().WithModel("claude-3-5-sonnet-20240620")

	return &Store{database: db, ctx: ctx, ai: antropicllm}, nil
}

func (s *Store) Close() error {
	db, err := s.database.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %v", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %v", err)
	}
	return nil
}

func (s *Store) GetProductByExternalID(externalID string) (*Product, error) {
	var product Product
	if err := s.database.First(&product, "external_id = ?", externalID).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			// Return a custom error if the product is not found
			return nil, fmt.Errorf("product not found: %v", err)
		}
		return nil, fmt.Errorf("failed to get product: %v", err)
	}

	return &product, nil
}

func (s *Store) GetReviewsByProductExternalID(externalID string) ([]Review, error) {
	var reviews []Review
	if err := s.database.Model(&Product{}).
		Select("reviews.*").
		Joins("left join reviews on reviews.product_id = products.id").
		Where("products.external_id = ?", externalID).
		Order("reviews.ia_generated desc").
		Scan(&reviews).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			// Return a custom error if the product is not found
			return nil, fmt.Errorf("product not found: %v", err)
		}
		return nil, fmt.Errorf("failed to get reviews: %v", err)
	}
	return reviews, nil
}

func (s *Store) GetProductAndReviewsByExternalID(externalID string) (*Product, error) {
	product, err := s.GetProductByExternalID(externalID)
	if err != nil {
		return nil, err
	}

	reviews, err := s.GetReviewsByProductExternalID(externalID)
	if err != nil {
		return nil, err
	}

	product.Reviews = reviews
	return product, nil
}

func (s *Store) CreateProduct(product *Product) (uint, error) {
	tx := s.database.WithContext(s.ctx)

	if result := tx.Create(product); result.Error != nil {
		return 0, fmt.Errorf("failed to create product: %v", result.Error)
	}

	return product.ID, nil
}

func (s *Store) CreateReview(review *Review) (uint, error) {
	tx := s.database.WithContext(s.ctx)

	if result := tx.Create(review); result.Error != nil {
		return 0, fmt.Errorf("failed to create review: %v", result.Error)
	}

	return review.ID, nil
}

func (s *Store) CreateSmartReview(externalID string) error {
	tx := s.database.WithContext(s.ctx)

	reviews, err := s.GetReviewsByProductExternalID(externalID)
	if err != nil {
		return err
	}

	if len(reviews) == 0 {
		return nil
	}

	// Check if reviews length is multiple of 5
	if (len(reviews)+1)%3 != 0 {
		return nil
	}

	// Iterate the reviews content and append to a single string
	var content string
	for _, review := range reviews {
		if !review.IAGenerated {
			content += "title:" + review.Title + " Review:" + review.Content + "\n"
		}
	}

	t := thread.New().AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent("Need a resume of this reviews in English with less than 200 characters, only the resume, don put anything else" + content),
		),
	)

	err = s.ai.Generate(context.Background(), t)
	if err != nil {
		panic(err)
	}

	summary := t.LastMessage().Contents[0].AsString()

	// Find if the AI review already exists
	var aiReview Review
	if err := s.database.First(&aiReview, "product_id = ? AND ia_generated = true", reviews[0].ProductID).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			aiReview = Review{
				ProductID:   reviews[0].ProductID,
				Author:      "AI Generated Summary Review",
				Title:       "AI Generated Summary Review",
				Content:     summary,
				Rating:      0, // Set the rating to 0
				IAGenerated: true,
			}
		}
	} else {
		aiReview.Content = summary
	}

	if result := tx.Save(&aiReview); result.Error != nil {
		return fmt.Errorf("failed to create review: %v", result.Error)
	}

	return nil
}

func (s *Store) GetProductsByShopID(shopID string) ([]Product, error) {
	var products []Product
	if err := s.database.Find(&products, "shop_id = ?", shopID).Error; err != nil {
		return nil, fmt.Errorf("failed to get products: %v", err)
	}

	return products, nil
}

func (s *Store) GetAverageRatingByProductExternalID(externalID string) (float32, error) {
	var avgRating float32
	if err := s.database.Model(&Review{}).
		Select("avg(rating)").
		Joins("left join products on reviews.product_id = products.id").
		Where("products.external_id = ? and ia_generated = false", externalID).
		Scan(&avgRating).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			// Return a custom error if the product is not found
			return 0, fmt.Errorf("product not found: %v", err)
		}
		return 0, fmt.Errorf("failed to get average rating: %v", err)
	}
	return avgRating, nil
}

func (s *Store) GetTotalReviewsQuantityByProductExternalID(externalID string) (int, error) {
	var totalReviews int
	if err := s.database.Model(&Review{}).
		Select("count(*)").
		Joins("left join products on reviews.product_id = products.id").
		Where("products.external_id = ? and ia_generated = false", externalID).
		Scan(&totalReviews).Error; err != nil {
		if gorm.ErrRecordNotFound == err {
			// Return a custom error if the product is not found
			return 0, fmt.Errorf("product not found: %v", err)
		}
		return 0, fmt.Errorf("failed to get total reviews quantity: %v", err)
	}
	return totalReviews, nil
}

func (s *Store) CreateReviewsKeywords(externalID string) error {
	// tx := s.database.WithContext(s.ctx)

	reviews, err := s.GetReviewsByProductExternalID(externalID)
	if err != nil {
		return err
	}

	if len(reviews) == 0 {
		return nil
	}

	// Check if reviews length is multiple of 5
	if (len(reviews)+1)%2 != 0 {
		return nil
	}

	// Iterate the reviews content and append to a single string
	var content string
	for _, review := range reviews {
		if !review.IAGenerated {
			content += review.Content + "\n"
		}
	}

	t := thread.New().AddMessage(
		thread.NewUserMessage().AddContent(
			thread.NewTextContent("Of this list of reviews separated by a new line, take at most the 5 most important " +
				"and repeated keywords and only return this in this format: word1,word2,word3,word4,word5" + content),
		),
	)

	err = s.ai.Generate(context.Background(), t)
	if err != nil {
		panic(err)
	}

	keywords := t.LastMessage().Contents[0].AsString()

	product, err := s.GetProductByExternalID(externalID)
	if err != nil {
		return err
	}

	product.Keywords = keywords

	// save the product with the keywords
	if result := s.database.Save(&product); result.Error != nil {
		return fmt.Errorf("failed to create review: %v", result.Error)
	}

	return nil
}
