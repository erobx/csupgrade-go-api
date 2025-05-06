package api

type StoreService interface {
	BuyCrate(crateID, userID string, amount int) (float64, []Item, error)
}

type StoreRepository interface {
	BuyCrate(crateID, userID string, amount int) (float64, []Item, error)
}

type storeService struct {
	storage StoreRepository
	logger LogService
}

func NewStoreService(storeRepo StoreRepository, logger LogService) StoreService {
	return &storeService{storage: storeRepo, logger: logger}
}

// Updates the user's current balance if they can purchase and adds skins to
// their inventory. Returns the new balance and items.
func (s *storeService) BuyCrate(crateID, userID string, amount int) (float64, []Item, error) {
	updatedBalance, addedItems, err := s.storage.BuyCrate(crateID, userID, amount)
	if err != nil {
		return updatedBalance, addedItems, err
	}

	s.logger.Info("successfully bought crate", "crate", crateID, "user", userID)

	return updatedBalance, addedItems, nil
}
