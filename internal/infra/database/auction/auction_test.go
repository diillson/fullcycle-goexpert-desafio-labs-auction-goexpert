package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetupTestDatabase(t *testing.T) (*mongo.Database, func()) {
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

func TestAutomaticAuctionClosing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Definir um timeout curto para o teste
	os.Setenv("AUCTION_INTERVAL", "3s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := SetupTestDatabase(t)
	defer cleanup()

	repo := NewAuctionRepository(db)
	defer repo.Cleanup()

	// Criar leilão com timestamp no passado para que expire rapidamente
	pastTime := time.Now().Add(-1 * time.Minute)
	auction := &auction_entity.Auction{
		Id:          "test-auction-1",
		ProductName: "Test Product",
		Category:    "Test Category",
		Description: "Test Description is longer than 10 chars",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   pastTime,
	}

	err := repo.CreateAuction(context.Background(), auction)
	assert.Nil(t, err)

	// Aguardar tempo suficiente para o leilão fechar
	t.Log("Waiting for auction to close automatically...")
	time.Sleep(10 * time.Second)

	// Verificar se o leilão foi fechado
	auctionFromDB, err := repo.FindAuctionById(context.Background(), auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, auctionFromDB.Status, "Auction should be automatically closed")
}
