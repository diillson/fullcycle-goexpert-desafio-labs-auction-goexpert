package bid

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/entity/bid_entity"
	"fullcycle-auction_go/internal/entity/user_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"fullcycle-auction_go/internal/infra/database/user"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDatabase(t *testing.T) (*mongo.Database, func()) {
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://admin:admin@mongodb-test:27017/auctions_test?authSource=admin"
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		dbName = "auctions_test"
	}

	db := client.Database(dbName)

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return db, cleanup
}

func TestBidCreationAndRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configurar os repositórios
	userRepo := user.NewUserRepository(db)
	auctionRepo := auction.NewAuctionRepository(db)
	bidRepo := NewBidRepository(db, auctionRepo)

	// 1. Criar um usuário
	userId := uuid.New().String()
	testUser := &user_entity.User{
		Id:   userId,
		Name: "Bidder Test User",
	}
	err := userRepo.CreateUser(context.Background(), testUser)
	assert.Nil(t, err, "Should create user without errors")

	// 2. Criar um leilão
	auctionId := uuid.New().String()
	testAuction := &auction_entity.Auction{
		Id:          auctionId,
		ProductName: "Test Product",
		Category:    "Test Category",
		Description: "Test Description is longer than 10 chars",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}
	err = auctionRepo.CreateAuction(context.Background(), testAuction)
	assert.Nil(t, err, "Should create auction without errors")

	// 3. Criar alguns lances
	bid1 := bid_entity.Bid{
		Id:        uuid.New().String(),
		UserId:    userId,
		AuctionId: auctionId,
		Amount:    100.0,
		Timestamp: time.Now(),
	}

	bid2 := bid_entity.Bid{
		Id:        uuid.New().String(),
		UserId:    userId,
		AuctionId: auctionId,
		Amount:    150.0,
		Timestamp: time.Now().Add(1 * time.Minute),
	}

	bids := []bid_entity.Bid{bid1, bid2}

	err = bidRepo.CreateBid(context.Background(), bids)
	assert.Nil(t, err, "Should create bids without errors")

	// 4. Encontrar lances por ID do leilão
	foundBids, err := bidRepo.FindBidByAuctionId(context.Background(), auctionId)
	assert.Nil(t, err, "Should find bids without errors")
	assert.Equal(t, 2, len(foundBids), "Should find 2 bids")

	// 5. Verificar lance vencedor (maior valor)
	winningBid, err := bidRepo.FindWinningBidByAuctionId(context.Background(), auctionId)
	assert.Nil(t, err, "Should find winning bid without errors")
	assert.Equal(t, 150.0, winningBid.Amount, "Winning bid should have the highest amount")
}

func TestBidRejectionForClosedAuction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Configurar os repositórios
	userRepo := user.NewUserRepository(db)
	auctionRepo := auction.NewAuctionRepository(db)
	bidRepo := NewBidRepository(db, auctionRepo)

	// 1. Criar um usuário
	userId := uuid.New().String()
	testUser := &user_entity.User{
		Id:   userId,
		Name: "Bidder Test User",
	}
	err := userRepo.CreateUser(context.Background(), testUser)
	assert.Nil(t, err)

	// 2. Criar um leilão já fechado
	auctionId := uuid.New().String()
	testAuction := &auction_entity.Auction{
		Id:          auctionId,
		ProductName: "Closed Test Product",
		Category:    "Test Category",
		Description: "Test Description is longer than 10 chars",
		Condition:   auction_entity.New,
		Status:      auction_entity.Completed, // Leilão já fechado
		Timestamp:   time.Now().Add(-30 * time.Minute),
	}
	err = auctionRepo.CreateAuction(context.Background(), testAuction)
	assert.Nil(t, err)

	// 3. Tentar criar um lance em um leilão fechado
	bid := bid_entity.Bid{
		Id:        uuid.New().String(),
		UserId:    userId,
		AuctionId: auctionId,
		Amount:    100.0,
		Timestamp: time.Now(),
	}

	// O lance é adicionado à fila, mas deve ser rejeitado internamente
	err = bidRepo.CreateBid(context.Background(), []bid_entity.Bid{bid})
	assert.Nil(t, err) // O método não retorna erro, apenas rejeita silenciosamente

	// Verificar que o lance não foi registrado
	foundBids, err := bidRepo.FindBidByAuctionId(context.Background(), auctionId)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(foundBids), "No bids should be accepted for closed auction")
}
