package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection     *mongo.Collection
	auctionTimeout time.Duration
	auctionsMutex  sync.RWMutex
	activeAuctions map[string]time.Time // mapa de leilões ativos e seus timestamps
	closeChan      chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ctx, cancel := context.WithCancel(context.Background())
	repo := &AuctionRepository{
		Collection:     database.Collection("auctions"),
		auctionTimeout: getAuctionTimeout(),
		activeAuctions: make(map[string]time.Time),
		auctionsMutex:  sync.RWMutex{},
		closeChan:      make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Carrega leilões ativos existentes no banco de dados
	go repo.loadExistingActiveAuctions(ctx)

	// Inicia a goroutine que verificará os leilões expirados
	go repo.checkExpiredAuctions()

	return repo
}

// loadExistingActiveAuctions carrega leilões ativos existentes no banco de dados
func (ar *AuctionRepository) loadExistingActiveAuctions(ctx context.Context) {
	// Criar filtro para leilões ativos
	filter := bson.M{"status": auction_entity.Active}

	// Encontrar todos os leilões ativos
	cursor, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error trying to load existing active auctions", err)
		return
	}
	defer cursor.Close(ctx)

	var auctionsMongo []AuctionEntityMongo
	if err := cursor.All(ctx, &auctionsMongo); err != nil {
		logger.Error("Error trying to decode active auctions", err)
		return
	}

	// Adicionar leilões ao mapa
	now := time.Now()
	ar.auctionsMutex.Lock()
	defer ar.auctionsMutex.Unlock()

	for _, auction := range auctionsMongo {
		// Calcular quando o leilão deve terminar
		creationTime := time.Unix(auction.Timestamp, 0)
		endTime := creationTime.Add(ar.auctionTimeout)

		// Se o leilão já expirou, marque-o para fechamento imediato
		// Caso contrário, adicione-o ao mapa com seu tempo de expiração
		if now.After(endTime) {
			// Leilão já deveria estar fechado, agende para fechamento imediato
			// Usando um tempo ligeiramente no futuro para permitir que o sistema inicialize
			ar.activeAuctions[auction.Id] = now.Add(5 * time.Second)
			logger.Info(fmt.Sprintf("Found expired auction %s, scheduling for immediate closure", auction.Id))
		} else {
			// Leilão ainda está ativo, adicione com seu tempo de expiração normal
			ar.activeAuctions[auction.Id] = endTime
			logger.Info(fmt.Sprintf("Loaded active auction %s, will expire at %s", auction.Id, endTime.Format(time.RFC3339)))
		}
	}

	logger.Info(fmt.Sprintf("Loaded %d existing active auctions", len(auctionsMongo)))
}

// getAuctionTimeout obtém o tempo de duração do leilão a partir de variáveis de ambiente
func getAuctionTimeout() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		logger.Error("Error parsing AUCTION_INTERVAL, using default value of 5 minutes", err)
		return time.Minute * 5
	}
	return duration
}

func getAuctionCheckInterval() time.Duration {
	checkInterval := os.Getenv("AUCTION_CHECK_INTERVAL")
	duration, err := time.ParseDuration(checkInterval)
	if err != nil {
		logger.Error("Error parsing AUCTION_CHECK_INTERVAL, using default value of 10 seconds", err)
		return time.Second * 10
	}
	return duration
}

// checkExpiredAuctions verifica periodicamente os leilões expirados e os fecha
func (ar *AuctionRepository) checkExpiredAuctions() {
	ticker := time.NewTicker(getAuctionCheckInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var expiredAuctions []string

			// Identificar leilões expirados (com lock de leitura)
			ar.auctionsMutex.RLock()
			for auctionID, endTime := range ar.activeAuctions {
				if now.After(endTime) {
					expiredAuctions = append(expiredAuctions, auctionID)
				}
			}
			ar.auctionsMutex.RUnlock()

			// Fechar leilões expirados
			for _, auctionID := range expiredAuctions {
				if err := ar.closeAuction(ar.ctx, auctionID); err != nil {
					logger.Error(fmt.Sprintf("Failed to close auction %s", auctionID), err)
				} else {
					// Remover do mapa após fechar com sucesso (com lock de escrita)
					ar.auctionsMutex.Lock()
					delete(ar.activeAuctions, auctionID)
					ar.auctionsMutex.Unlock()
					logger.Info(fmt.Sprintf("Auction %s closed automatically", auctionID))
				}
			}

		case <-ar.closeChan:
			return
		}
	}
}

// closeAuction atualiza o status do leilão para completo no banco de dados
func (ar *AuctionRepository) closeAuction(ctx context.Context, auctionID string) *internal_error.InternalError {
	filter := bson.M{"_id": auctionID}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

	_, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error(fmt.Sprintf("Error closing auction %s", auctionID), err)
		return internal_error.NewInternalServerError(fmt.Sprintf("Error closing auction %s", auctionID))
	}

	return nil
}

// Cleanup encerra as goroutines e recursos associados
func (ar *AuctionRepository) Cleanup() {
	close(ar.closeChan)
	ar.cancel()
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	// Registra o leilão no mapa de leilões ativos com seu tempo de expiração
	endTime := auctionEntity.Timestamp.Add(ar.auctionTimeout)

	ar.auctionsMutex.Lock()
	ar.activeAuctions[auctionEntity.Id] = endTime
	ar.auctionsMutex.Unlock()

	logger.Info(fmt.Sprintf("Auction %s created, will expire at %s", auctionEntity.Id, endTime.Format(time.RFC3339)))

	return nil
}

// FindActiveAuctions retorna todos os leilões ativos (para testes)
func (ar *AuctionRepository) FindActiveAuctions(ctx context.Context) ([]auction_entity.Auction, *internal_error.InternalError) {
	filter := bson.M{"status": auction_entity.Active}

	cursor, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error trying to find active auctions", err)
		return nil, internal_error.NewInternalServerError("Error trying to find active auctions")
	}
	defer cursor.Close(ctx)

	var auctions []auction_entity.Auction
	var auctionsMongo []AuctionEntityMongo

	if err := cursor.All(ctx, &auctionsMongo); err != nil {
		logger.Error("Error trying to decode auctions", err)
		return nil, internal_error.NewInternalServerError("Error trying to decode auctions")
	}

	for _, a := range auctionsMongo {
		auctions = append(auctions, auction_entity.Auction{
			Id:          a.Id,
			ProductName: a.ProductName,
			Category:    a.Category,
			Description: a.Description,
			Condition:   a.Condition,
			Status:      a.Status,
			Timestamp:   time.Unix(a.Timestamp, 0),
		})
	}

	return auctions, nil
}
