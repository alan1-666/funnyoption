package custody

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type AddressMapping struct {
	UserID  int64
	Chain   string
	Network string
	Coin    string
	Address string
	KeyID   string
}

func (s *Store) LookupUserByAddress(ctx context.Context, address, chain, network string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id FROM custody_address_mapping
		WHERE address = $1 AND chain = $2 AND network = $3
		LIMIT 1
	`, norm(address), norm(chain), norm(network)).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return userID, nil
}

func (s *Store) GetUserAddress(ctx context.Context, userID int64, chain, network, coin string) (string, error) {
	var address string
	err := s.db.QueryRowContext(ctx, `
		SELECT address FROM custody_address_mapping
		WHERE user_id = $1 AND chain = $2 AND network = $3 AND coin = $4
		LIMIT 1
	`, userID, norm(chain), norm(network), norm(coin)).Scan(&address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return address, nil
}

func (s *Store) GetUserAddressWithKeyID(ctx context.Context, userID int64, chain, network, coin string) (address, keyID string, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT address, COALESCE(key_id, '') FROM custody_address_mapping
		WHERE user_id = $1 AND chain = $2 AND network = $3 AND coin = $4
		LIMIT 1
	`, userID, norm(chain), norm(network), norm(coin)).Scan(&address, &keyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil
		}
		return "", "", err
	}
	return address, keyID, nil
}

func (s *Store) SaveAddressMapping(ctx context.Context, m AddressMapping) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO custody_address_mapping (user_id, tenant_id, chain, network, coin, address, key_id)
		VALUES ($1, 'funnyoption', $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, chain, network, coin, address) DO UPDATE SET key_id = EXCLUDED.key_id
	`, m.UserID, norm(m.Chain), norm(m.Network), norm(m.Coin), norm(m.Address), m.KeyID)
	return err
}

func (s *Store) IsDepositProcessed(ctx context.Context, bizID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM custody_deposits WHERE biz_id = $1)
	`, bizID).Scan(&exists)
	return exists, err
}

type DepositRecord struct {
	BizID        string
	UserID       int64
	Address      string
	Asset        string // original deposit coin (e.g. "BNB")
	CreditAsset  string // credited asset (always "USDT")
	ChainAmount  string
	CreditAmount int64
	ChainID      int64
	TxHash       string
	TxIndex      int
}

func (s *Store) InsertDeposit(ctx context.Context, d DepositRecord) error {
	creditAsset := strings.ToUpper(strings.TrimSpace(d.CreditAsset))
	if creditAsset == "" {
		creditAsset = strings.ToUpper(strings.TrimSpace(d.Asset))
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO custody_deposits (biz_id, user_id, address, asset, credit_asset, chain_amount, credit_amount, chain_id, tx_hash, tx_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (biz_id) DO NOTHING
	`, d.BizID, d.UserID, norm(d.Address), strings.ToUpper(strings.TrimSpace(d.Asset)),
		creditAsset, d.ChainAmount, d.CreditAmount, d.ChainID, d.TxHash, d.TxIndex)
	return err
}

func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
