package client

import (
	"context"
	"fmt"
	"time"

	accountv1 "funnyoption/internal/gen/accountv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FreezeRequest struct {
	UserID  int64
	Asset   string
	RefType string
	RefID   string
	Amount  int64
}

type FreezeRecord struct {
	FreezeID string
	UserID   int64
	Asset    string
	RefType  string
	RefID    string
	Amount   int64
}

type Balance struct {
	UserID    int64
	Asset     string
	Available int64
	Frozen    int64
	Total     int64
}

type CreditResult struct {
	UserID    int64
	Asset     string
	Available int64
	Frozen    int64
	Total     int64
	Applied   bool
}

type DebitResult struct {
	UserID    int64
	Asset     string
	Available int64
	Frozen    int64
	Total     int64
	Applied   bool
}

type AccountClient interface {
	PreFreeze(ctx context.Context, req FreezeRequest) (FreezeRecord, error)
	ReleaseFreeze(ctx context.Context, freezeID string) error
	GetBalance(ctx context.Context, userID int64, asset string) (Balance, error)
	CreditBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (CreditResult, error)
	DebitBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (DebitResult, error)
	Close() error
}

type GRPCClient struct {
	conn    *grpc.ClientConn
	client  accountv1.AccountServiceClient
	timeout time.Duration
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	if target == "" {
		return nil, fmt.Errorf("account grpc target is empty")
	}

	conn, err := grpc.Dial(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		conn:    conn,
		client:  accountv1.NewAccountServiceClient(conn),
		timeout: 3 * time.Second,
	}, nil
}

func (c *GRPCClient) PreFreeze(ctx context.Context, req FreezeRequest) (FreezeRecord, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.PreFreeze(callCtx, &accountv1.PreFreezeRequest{
		UserId:  req.UserID,
		Asset:   req.Asset,
		RefType: req.RefType,
		RefId:   req.RefID,
		Amount:  req.Amount,
	})
	if err != nil {
		return FreezeRecord{}, err
	}

	return FreezeRecord{
		FreezeID: resp.GetFreezeId(),
		UserID:   resp.GetUserId(),
		Asset:    resp.GetAsset(),
		RefType:  resp.GetRefType(),
		RefID:    resp.GetRefId(),
		Amount:   resp.GetAmount(),
	}, nil
}

func (c *GRPCClient) ReleaseFreeze(ctx context.Context, freezeID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.ReleaseFreeze(callCtx, &accountv1.ReleaseFreezeRequest{
		FreezeId: freezeID,
	})
	return err
}

func (c *GRPCClient) GetBalance(ctx context.Context, userID int64, asset string) (Balance, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetBalance(callCtx, &accountv1.GetBalanceRequest{
		UserId: userID,
		Asset:  asset,
	})
	if err != nil {
		return Balance{}, err
	}

	return Balance{
		UserID:    resp.GetUserId(),
		Asset:     resp.GetAsset(),
		Available: resp.GetAvailable(),
		Frozen:    resp.GetFrozen(),
		Total:     resp.GetTotal(),
	}, nil
}

func (c *GRPCClient) CreditBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (CreditResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CreditBalance(callCtx, &accountv1.CreditBalanceRequest{
		UserId:  userID,
		Asset:   asset,
		Amount:  amount,
		RefType: refType,
		RefId:   refID,
	})
	if err != nil {
		return CreditResult{}, err
	}

	return CreditResult{
		UserID:    resp.GetUserId(),
		Asset:     resp.GetAsset(),
		Available: resp.GetAvailable(),
		Frozen:    resp.GetFrozen(),
		Total:     resp.GetTotal(),
		Applied:   resp.GetApplied(),
	}, nil
}

func (c *GRPCClient) DebitBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (DebitResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.DebitBalance(callCtx, &accountv1.DebitBalanceRequest{
		UserId:  userID,
		Asset:   asset,
		Amount:  amount,
		RefType: refType,
		RefId:   refID,
	})
	if err != nil {
		return DebitResult{}, err
	}

	return DebitResult{
		UserID:    resp.GetUserId(),
		Asset:     resp.GetAsset(),
		Available: resp.GetAvailable(),
		Frozen:    resp.GetFrozen(),
		Total:     resp.GetTotal(),
		Applied:   resp.GetApplied(),
	}, nil
}

func (c *GRPCClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
