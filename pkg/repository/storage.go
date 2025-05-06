package repository

import (
	"context"
	"errors"
	"log"
	"math/rand/v2"
	"strings"

	"github.com/erobx/csupgrade-go-api/pkg/api"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Contains methods for interacting with the DB
type Storage interface {
	// User updates
	CreateUser(request *api.NewUserRequest) (string, error)
	GetUserByID(userID string) (api.User, error)
	GetUserAndHashByEmail(email string) (api.User, string, error)
	GetInventory(userID string) (api.Inventory, error)
	GetRecentTradeups(userID string) ([]api.RecentTradeup, error)
	GetRecentWinnings(userID string) ([]api.Item, error)

	// Store
	BuyCrate(crateID, userID string, amount int) (float64, []api.Item, error)

	// Tradeups
	GetAllTradeups() ([]api.Tradeup, error)
	GetTradeupByID(tradeupID string) (api.Tradeup, error)
	AddSkinToTradeup(tradeupID, invID string) error
	RemoveSkinFromTradeup(tradeupID, invID string) error
	MaintainTradeupCount() error
	GetUserContribution(tradeupID, userID string) (int, error)

	// Helpers
	CheckSkinOwnership(invID, userID string) (bool, error)
	IsTradeupFull(tradeupID string) (bool, error)
	StartTimer(tradeupID string) error
	StopTimer(tradeupID string) error
	GetStatus(tradeupID string) (string, error)
	SetStatus(tradeupID, status string) error
	GetExpired() ([]api.Tradeup, error)
	DetermineWinner(tradeupID int) (string, error)
	GiveNewItem(userID, rarity string, avgFloat float64) (api.Item, error)
}

type storage struct {
	db     *pgxpool.Pool
	cdnUrl string
}

func NewStorage(db *pgxpool.Pool, url string) Storage {
	return &storage{db: db, cdnUrl: url}
}

func (s *storage) BuyCrate(crateID, userID string, amount int) (float64, []api.Item, error) {
	var updatedBalance float64
	var addedItems []api.Item

	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return updatedBalance, addedItems, err
	}
	defer func() {
		tx.Commit(context.Background())
	}()

	q := `
	with updated as (
		update users
		set balance = balance - ((select cost from crates where id=$1) * $2)
		where id = (select id from users where id=$3)
		and balance >= ((select cost from crates where id=$1) * $2)
		returning balance
	) select * from updated
	`
	err = tx.QueryRow(context.Background(), q, crateID, amount, userID).Scan(&updatedBalance)
	if err != nil {
		tx.Rollback(context.Background())
		return updatedBalance, addedItems, errors.New("insufficent funds")
	}

	q = `select skin_id from crate_skins where crate_id=$1 order by random() limit $2`
	rows, err := tx.Query(context.Background(), q, crateID, amount)
	if err != nil {
		tx.Rollback(context.Background())
		return updatedBalance, addedItems, err
	}
	defer rows.Close()

	var skinIDs []int
	for rows.Next() {
		var skinID int
		err := rows.Scan(&skinID)
		if err != nil {
			log.Println("Failed scanning skin id")
			tx.Rollback(context.Background())
			return updatedBalance, addedItems, err
		}

		skinIDs = append(skinIDs, skinID)
	}

	for _, skinID := range skinIDs {
		var canBeStatTrak, isStatTrak bool
		q = "select can_be_stattrak from skins where id=$1"
		err = tx.QueryRow(context.Background(), q, skinID).Scan(&canBeStatTrak)

		if err != nil {
			log.Println("Failed scanning StatTrak")
			tx.Rollback(context.Background())
			return updatedBalance, addedItems, err
		}

		if canBeStatTrak {
			isStatTrak = api.IsStatTrak()
		}

		// generate float value and corresponding wear name
		floatValue := rand.Float64()
		wear := api.GetWearFromFloatValue(floatValue)

		var skin api.Skin
		var item api.Item
		var imageKey string

		// add skins for each randomly selected id
		q = `
		with item as (
			insert into inventory(user_id,skin_id,wear_str,wear_num,price,is_stattrak,created_at) 
			values($1,$2,$3,$4,12.34,$5,now())
			returning *
		) select item.id, item.skin_id, item.wear_str, item.wear_num, item.price, 
			item.is_stattrak, item.was_won, item.created_at, item.visible, s.name, 
			s.rarity, s.collection, s.image_key
		from item
		join skins s on s.id = item.skin_id
		`
		row := tx.QueryRow(context.Background(), q, userID, skinID, wear, floatValue, isStatTrak)
		err = row.Scan(&item.InvID, &skin.ID, &skin.Wear, &skin.Float, &skin.Price,
			&skin.IsStatTrak, &skin.WasWon, &skin.CreatedAt, &item.Visible, &skin.Name,
			&skin.Rarity, &skin.Collection, &imageKey)

		if err != nil {
			log.Println("Failed scanning item")
			tx.Rollback(context.Background())
			return updatedBalance, addedItems, err
		}

		skin.ImgSrc = s.createImgSrc(imageKey)
		item.Data = skin
		addedItems = append(addedItems, item)
	}

	return updatedBalance, addedItems, nil
}

// url + guns/ak/imageKey
func (s *storage) createImgSrc(imageKey string) string {
	prefix := imageKey[:strings.Index(imageKey, "-")]
	url := s.cdnUrl + "guns/" + prefix + "/" + imageKey
	return url
}
