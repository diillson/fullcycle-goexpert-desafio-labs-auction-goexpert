package integration

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/entity/bid_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"fullcycle-auction_go/internal/infra/database/bid"
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

func TestSimpleAuctionLifecycleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Configurar um intervalo de leilão curto para testes
	os.Setenv("AUCTION_INTERVAL", "3s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	// Configurar o banco de dados
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Inicializar repositórios
	auctionRepo := auction.NewAuctionRepository(db)
	bidRepo := bid.NewBidRepository(db, auctionRepo)
	defer auctionRepo.Cleanup()

	ctx := context.Background()

	// 1. Criar um leilão
	auctionId := uuid.New().String()
	testAuction := &auction_entity.Auction{
		Id:          auctionId,
		ProductName: "Integration Test Product",
		Category:    "Electronics",
		Description: "A product for integration testing",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	err := auctionRepo.CreateAuction(ctx, testAuction)
	assert.Nil(t, err)

	// 2. Verificar que o leilão foi criado com status ativo
	createdAuction, err := auctionRepo.FindAuctionById(ctx, auctionId)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Active, createdAuction.Status)

	// 3. Criar um lance para o leilão
	bidId := uuid.New().String()
	userId := uuid.New().String()

	testBid := bid_entity.Bid{
		Id:        bidId,
		UserId:    userId,
		AuctionId: auctionId,
		Amount:    100.0,
		Timestamp: time.Now(),
	}

	err = bidRepo.CreateBid(ctx, []bid_entity.Bid{testBid})
	assert.Nil(t, err)

	// 4. Esperar o leilão fechar automaticamente
	t.Log("Waiting for auction to close automatically...")
	time.Sleep(10 * time.Second)

	// 5. Verificar que o leilão foi fechado
	updatedAuction, err := auctionRepo.FindAuctionById(ctx, auctionId)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, updatedAuction.Status, "Auction should be closed automatically")

	// 6. Verificar que o lance vencedor pode ser encontrado
	winningBid, err := bidRepo.FindWinningBidByAuctionId(ctx, auctionId)
	assert.Nil(t, err)
	assert.Equal(t, bidId, winningBid.Id)
	assert.Equal(t, 100.0, winningBid.Amount)
}
