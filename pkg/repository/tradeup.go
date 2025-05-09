package repository

import (
	"context"
	"math/rand/v2"

	"github.com/erobx/csupgrade-go-api/pkg/api"
	"github.com/jackc/pgx/v5"
)

func (s *storage) GetAllTradeups() ([]api.Tradeup, error) {
	var tradeups []api.Tradeup
	var ids []string

	q := `select id from tradeups where current_status = 'Active' or current_status = 'Waiting'`
	rows, err := s.db.Query(context.Background(), q)
	if err != nil {
		return tradeups, nil
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return tradeups, err
		}

		ids = append(ids, id)
	}

	for _, id := range ids {
		t, err := s.GetTradeupByID(id)
		if err != nil {
			return tradeups, err
		}

		tradeups = append(tradeups, t)
	}

	return tradeups, nil
}

func (s *storage) GetTradeupByID(tradeupID string) (api.Tradeup, error) {
	var tradeup api.Tradeup
	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return tradeup, err
	}
	defer func() {
		tx.Commit(context.Background())
	}()

	q := `
	select id, rarity, current_status, stop_time, mode from tradeups where id=$1
	`
	err = s.db.QueryRow(context.Background(), q, tradeupID).Scan(&tradeup.ID,
		&tradeup.Rarity, &tradeup.Status, &tradeup.StopTime, &tradeup.Mode)
	if err != nil {
		tx.Rollback(context.Background())
		return tradeup, err
	}

	//if winner.Valid {
	//	tradeup.Winner = winner.String
	//} else {
	//	tradeup.Winner = ""
	//}

	items := make([]api.Item, 0)
	players := make(map[string]api.Player, 0)
	q = `
	select i.id, i.skin_id, i.wear_str, i.wear_num, i.price, i.is_stattrak, 
		u.username, u.avatar_key, s.name, s.rarity, s.collection, s.image_key
	from tradeups t
	join tradeups_skins ts on ts.tradeup_id = t.id
	join inventory i on i.id = ts.inv_id
	join users u on u.id = i.user_id
	join skins s on s.id = i.skin_id
	where t.id=$1
	`
	rows, err := s.db.Query(context.Background(), q, tradeupID)
	if err != nil {
		tx.Rollback(context.Background())
		return tradeup, err
	}
	defer rows.Close()

	for rows.Next() {
		var item api.Item
		var skin api.Skin
		var player api.Player
		var avatarKey string
		var imageKey string

		err := rows.Scan(&item.InvID, &skin.ID, &skin.Wear, &skin.Float, &skin.Price,
			&skin.IsStatTrak, &player.Username, &avatarKey, &skin.Name, &skin.Rarity, &skin.Collection, &imageKey)
		if err != nil {
			tx.Rollback(context.Background())
			return tradeup, err
		}

		if _, ok := players[player.Username]; !ok {
			player.AvatarSrc = avatarKey
			players[player.Username] = player
		}

		skin.ImgSrc = s.createImgSrc(imageKey)
		item.Data = skin
		items = append(items, item)
	}

	for _, p := range players {
		tradeup.Players = append(tradeup.Players, p)
	}

	tradeup.Items = items
	return tradeup, nil
}

func (s *storage) IsTradeupFull(tradeupID string) (bool, error) {
	var count int
	q := "select count(*) from tradeups_skins where tradeup_id=$1"
	err := s.db.QueryRow(context.Background(), q, tradeupID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count == 10 {
		return true, nil
	}

	return false, nil
}

func (s *storage) AddSkinToTradeup(tradeupID, invID string) error {
	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		tx.Commit(context.Background())
	}()

	q := "insert into tradeups_skins values($1,$2)"
	_, err = tx.Exec(context.Background(), q, tradeupID, invID)
	if err != nil {
		tx.Rollback(context.Background())
		return err
	}

	q = "update inventory set visible=false where id=$1"
	_, err = tx.Exec(context.Background(), q, invID)
	if err != nil {
		tx.Rollback(context.Background())
		return err
	}

	return nil
}

func (s *storage) RemoveSkinFromTradeup(tradeupID, invID string) error {
	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		tx.Commit(context.Background())
	}()

	q := "delete from tradeups_skins where tradeup_id=$1 and inv_id=$2"
	_, err = tx.Exec(context.Background(), q, tradeupID, invID)
	if err != nil {
		tx.Rollback(context.Background())
		return err
	}

	q = "update inventory set visible=true where id=$1"
	_, err = tx.Exec(context.Background(), q, invID)
	if err != nil {
		tx.Rollback(context.Background())
		return err
	}

	return nil
}

func (s *storage) GetUserContribution(tradeupID, userID string) (int, error) {
	var contrib int

	q := `
	select count(ts.inv_id) as skin_count
	from tradeups_skins ts
	join inventory i on ts.inv_id = i.id
	where ts.tradeup_id = $1
	  and i.user_id = $2
	`
	err := s.db.QueryRow(context.Background(), q, tradeupID, userID).Scan(&contrib)
	if err != nil {
		return 0, err
	}

	return contrib, nil
}

func (s *storage) StartTimer(tradeupID string) error {
	// UTC timestamp is off by 4 hours currently for me
	q := "update tradeups set stop_time=now()+interval '1 min',current_status='Waiting' where id=$1"
	_, err := s.db.Exec(context.Background(), q, tradeupID)
	return err
}

func (s *storage) StopTimer(tradeupID string) error {
	q := "update tradeups set stop_time=now()+interval '5 year',current_status='Active' where id=$1"
	_, err := s.db.Exec(context.Background(), q, tradeupID)
	return err
}

func (s *storage) GetStatus(tradeupID string) (string, error) {
	var status string
	q := "select current_status from tradeups where id=$1"
	err := s.db.QueryRow(context.Background(), q, tradeupID).Scan(&status)
	return status, err
}

func (s *storage) SetStatus(tradeupID, status string) error {
	q := "update tradeups set current_status=$1 where id=$2"
	_, err := s.db.Exec(context.Background(), q, status, tradeupID)
	return err
}

// Returns all expired tradeups
func (s *storage) GetExpired() ([]api.Tradeup, error) {
	var expired []api.Tradeup

	q := `
	select id from tradeups where current_status='Waiting' and
	now() > stop_time
	`
	rows, err := s.db.Query(context.Background(), q)
	if err != nil {
		return expired, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return expired, err
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		t, err := s.GetTradeupByID(id)
		if err != nil {
			return expired, err
		}
		expired = append(expired, t)
	}

	return expired, nil
}

// Determines the winner for a specific tradeup and marks inventory items in
// the tradeup as used
func (s *storage) DetermineWinner(tradeupID int) (string, error) {
	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() {
		tx.Commit(context.Background())
	}()

	q := `
	select distinct i.user_id from tradeups_skins ts 
	join inventory i on i.id = ts.inv_id
	where ts.tradeup_id = $1
	`
	rows, err := tx.Query(context.Background(), q, tradeupID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var userID string
		err := rows.Scan(&userID)
		if err != nil {
			return "", err
		}
		playerIDs = append(playerIDs, userID)
	}

	playerWeights := make(map[string]int)

	for _, id := range playerIDs {
		var weight int
		q := `
		select count(ts.inv_id) as skin_count
		from tradeups_skins ts
		join inventory i on ts.inv_id = i.id
		where ts.tradeup_id = $1
		  and i.user_id = $2
		`
		err := tx.QueryRow(context.Background(), q, tradeupID, id).Scan(&weight)
		if err != nil {
			return "", err
		}

		playerWeights[id] = weight
	}

	var winner string
	randomNum := rand.IntN(100)
	currWeight := 0

	for player, weight := range playerWeights {
		currWeight += weight * 10
		if randomNum < currWeight {
			winner = player
			break
		}
	}

	q = `update tradeups set current_status='Completed', winner=$1 where id=$2`
	_, err = tx.Exec(context.Background(), q, winner, tradeupID)
	if err != nil {
		tx.Rollback(context.Background())
		return "", err
	}

	q = `
	update inventory
	set was_used = true
	from tradeups_skins
	where tradeups_skins.inv_id = inventory.id
		and tradeups_skins.tradeup_id = $1
	`
	_, err = tx.Exec(context.Background(), q, tradeupID)
	if err != nil {
		tx.Rollback(context.Background())
		return "", err
	}

	return winner, nil
}

// Gives user a new item of the requested rarity
func (s *storage) GiveNewItem(userID, rarity string, avgFloat float64) (api.Item, error) {
	var item api.Item
	var skin api.Skin
	var wearMin, wearMax float64
	var canBeStatTrak bool
	var imageKey string

	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return item, err
	}
	defer func() {
		tx.Commit(context.Background())
	}()

	q := "select id,name,rarity,collection,wear_min,wear_max,can_be_stattrak,image_key from skins where rarity=$1 order by random() limit 1"
	err = tx.QueryRow(context.Background(), q, rarity).Scan(&skin.ID, &skin.Name,
		&skin.Rarity, &skin.Collection, &wearMin, &wearMax, &canBeStatTrak, &imageKey)
	if err != nil {
		return item, err
	}

	wearNum := ((wearMax - wearMin) * avgFloat) + wearMin
	wearStr := api.GetWearNameFromFloat(wearNum)
	isStatTrak := false
	if canBeStatTrak {
		isStatTrak = api.IsStatTrak()
	}

	q = `
    insert into inventory(user_id, skin_id, wear_str, wear_num, price, is_stattrak, was_won)
	values ($1,$2,$3,$4,12.34,$5,true) 
	returning id,wear_str,wear_num,price,is_stattrak,was_won,created_at
    `

	err = tx.QueryRow(context.Background(), q, userID, skin.ID, wearStr, wearNum,
		isStatTrak).Scan(&item.InvID, &skin.Wear, &skin.Float, &skin.Price, &skin.IsStatTrak,
		&skin.WasWon, &skin.CreatedAt)
	if err != nil {
		tx.Rollback(context.Background())
		return item, err
	}

	skin.ImgSrc = s.createImgSrc(imageKey)
	item.Data = skin
	item.Visible = true

	return item, nil
}

func (s *storage) MaintainTradeupCount() error {
	rarities := []string{"Consumer", "Industrial", "Mil-Spec", "Restricted", "Classified"}
	tx, err := s.db.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}

	for _, r := range rarities {
		count := 0
		q := "select count(*) from tradeups where rarity=$1 and current_status in ('Active', 'Waiting')"
		if err := tx.QueryRow(context.Background(), q, r).Scan(&count); err != nil {
			return err
		}

		if count < 5 {
			q := "insert into tradeups(rarity) values($1)"
			_, err := tx.Exec(context.Background(), q, r)
			if err != nil {
				tx.Rollback(context.Background())
				return err
			}
		}
	}

	tx.Commit(context.Background())
	return nil
}
