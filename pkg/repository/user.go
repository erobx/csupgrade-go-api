package repository

import (
	"context"
	"encoding/json"

	"github.com/erobx/csupgrade/backend/pkg/api"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *storage) CreateUser(request *api.NewUserRequest) (string, error) {
	id := uuid.New().String()

	hashed, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	q := "insert into users(id,username,email,hash,avatar_key,created_at) values($1,$2,$3,$4,$5,now())"
	_, err = s.db.Exec(context.Background(), q, id, request.Username, request.Email, string(hashed), "none")

	return id, err
}

func (s *storage) GetUserByID(userID string) (api.User, error) {
	var user api.User
	var avatarKey string

	q := "select id,username,email,balance,refresh_token_version,avatar_key,created_at from users where id=$1"
	row := s.db.QueryRow(context.Background(), q, userID)
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Balance,
			&user.RefreshTokenVersion, &avatarKey, &user.CreatedAt)

	// TODO: generate url for avatarKey, then assign to user.AvatarSrc

	return user, err
}

func (s *storage) GetUserAndHashByEmail(email string) (api.User, string, error) {
	var user api.User
	var hash string
	var avatarKey string

	q := "select * from users where email=$1"
	row := s.db.QueryRow(context.Background(), q, email)
	err := row.Scan(&user.ID, &user.Username, &user.Email, &hash, &user.Balance,
			&user.RefreshTokenVersion, &avatarKey, &user.CreatedAt)

	// TODO: generate url for avatarKey, then assign to user.AvatarSrc

	return user, hash, err
}

func (s *storage) GetInventory(userID string) (api.Inventory, error) {
	inventory := api.Inventory{
		UserID: userID,
		Items: make([]api.Item, 0),
	}

	q := `
	select i.id, i.skin_id, i.wear_str, i.wear_num, i.price, i.is_stattrak,
		i.was_won, i.created_at, i.visible, s.name, s.rarity, s.collection, s.image_key
	from inventory i
	join skins s on s.id = i.skin_id
		where i.user_id = $1 and i.was_used = false
	`
	rows, err := s.db.Query(context.Background(), q, userID)
	if err != nil {
		return inventory, err
	}
	defer rows.Close()

	for rows.Next() {
		var item api.Item
		var skin api.Skin
		var imageKey string

		err := rows.Scan(&item.InvID, &skin.ID, &skin.Wear, &skin.Float, &skin.Price,
						&skin.IsStatTrak, &skin.WasWon, &skin.CreatedAt, &item.Visible,
						&skin.Name, &skin.Rarity, &skin.Collection, &imageKey)
		if err != nil {
			return inventory, err
		}

		skin.ImgSrc = s.createImgSrc(imageKey)
		item.Data = skin
		inventory.Items = append(inventory.Items, item)
	}

	return inventory, nil
}

func (s *storage) CheckSkinOwnership(invID, userID string) (bool, error) {
	isOwned := false

	q := "select exists(select 1 from inventory where id=$1 and user_id=$2)"
	err := s.db.QueryRow(context.Background(), q, invID, userID).Scan(&isOwned)

	return isOwned, err
}

/*
type Row = {
  tradeupId: string;
  rarity: string;
  status: string;
  skins: Skin[];
  value: number;
}
*/

func (s *storage) GetRecentTradeups(userID string) ([]api.RecentTradeup, error) {
	var recentTradeups []api.RecentTradeup

	q := `
    SELECT
        t.id AS tradeup_id,
        t.rarity,
        t.current_status AS status,
        t.mode,
        MAX(ts.entered) AS last_entered,
        JSONB_AGG(
			JSONB_BUILD_OBJECT(
				'invId', i.id,
				'data', JSONB_BUILD_OBJECT(
					'name', s.name,
					'wear', i.wear_str,
					'price', i.price
				),
				'visible', true
			)
		) AS items
    FROM 
        tradeups t
    INNER JOIN 
        tradeups_skins ts ON t.id = ts.tradeup_id
    INNER JOIN 
        inventory i ON ts.inv_id = i.id
    INNER JOIN 
        skins s ON i.skin_id = s.id
    WHERE 
        i.user_id = $1
    GROUP BY 
        t.id, t.rarity, t.current_status, t.stop_time, t.mode
    ORDER BY 
        last_entered DESC
	LIMIT 5
    `

	rows, err := s.db.Query(context.Background(), q, userID)
	if err != nil {
		return recentTradeups, err
	}
	defer rows.Close()

	for rows.Next() {
		var t api.RecentTradeup
		var itemsJSON string

		if err := rows.Scan(
			&t.ID, &t.Rarity, &t.Status, &t.Mode, &t.LastEntered, &itemsJSON);
		err != nil {
			return recentTradeups, err
		}

		if err := json.Unmarshal([]byte(itemsJSON), &t.Items); err != nil {
			return nil, err
		}
		recentTradeups = append(recentTradeups, t)
	}

	return recentTradeups, nil
}

func (s *storage) GetRecentWinnings(userID string) ([]api.Item, error) {
	var winnings []api.Item



	return winnings, nil
}
