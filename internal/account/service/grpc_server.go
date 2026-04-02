package service

import (
	"context"
	"errors"

	accountv1 "funnyoption/internal/gen/accountv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	accountv1.UnimplementedAccountServiceServer
	book *BalanceBook
}

func NewGRPCServer(book *BalanceBook) *GRPCServer {
	return &GRPCServer{book: book}
}

func (s *GRPCServer) PreFreeze(ctx context.Context, req *accountv1.PreFreezeRequest) (*accountv1.PreFreezeResponse, error) {
	_ = ctx

	record, err := s.book.PreFreeze(FreezeRequest{
		UserID:  req.GetUserId(),
		Asset:   req.GetAsset(),
		RefType: req.GetRefType(),
		RefID:   req.GetRefId(),
		Amount:  req.GetAmount(),
	})
	if err != nil {
		return nil, mapBalanceBookError(err)
	}

	return &accountv1.PreFreezeResponse{
		FreezeId: record.FreezeID,
		UserId:   record.UserID,
		Asset:    record.Asset,
		RefType:  record.RefType,
		RefId:    record.RefID,
		Amount:   record.Amount,
	}, nil
}

func (s *GRPCServer) ReleaseFreeze(ctx context.Context, req *accountv1.ReleaseFreezeRequest) (*accountv1.ReleaseFreezeResponse, error) {
	_ = ctx

	if err := s.book.ReleaseFreeze(req.GetFreezeId()); err != nil {
		return nil, mapBalanceBookError(err)
	}
	return &accountv1.ReleaseFreezeResponse{
		FreezeId: req.GetFreezeId(),
	}, nil
}

func (s *GRPCServer) GetBalance(ctx context.Context, req *accountv1.GetBalanceRequest) (*accountv1.GetBalanceResponse, error) {
	_ = ctx

	balance := s.book.GetBalance(req.GetUserId(), req.GetAsset())
	return &accountv1.GetBalanceResponse{
		UserId:    balance.UserID,
		Asset:     balance.Asset,
		Available: balance.Available,
		Frozen:    balance.Frozen,
		Total:     balance.Total(),
	}, nil
}

func (s *GRPCServer) CreditBalance(ctx context.Context, req *accountv1.CreditBalanceRequest) (*accountv1.CreditBalanceResponse, error) {
	_ = ctx

	balance, applied, err := s.book.CreditAvailableWithRef(CreditRequest{
		UserID:  req.GetUserId(),
		Asset:   req.GetAsset(),
		Amount:  req.GetAmount(),
		RefType: req.GetRefType(),
		RefID:   req.GetRefId(),
	})
	if err != nil {
		return nil, mapBalanceBookError(err)
	}

	return &accountv1.CreditBalanceResponse{
		UserId:    balance.UserID,
		Asset:     balance.Asset,
		Available: balance.Available,
		Frozen:    balance.Frozen,
		Total:     balance.Total(),
		Applied:   applied,
	}, nil
}

func (s *GRPCServer) DebitBalance(ctx context.Context, req *accountv1.DebitBalanceRequest) (*accountv1.DebitBalanceResponse, error) {
	_ = ctx

	balance, applied, err := s.book.DebitAvailableWithRef(DebitRequest{
		UserID:  req.GetUserId(),
		Asset:   req.GetAsset(),
		Amount:  req.GetAmount(),
		RefType: req.GetRefType(),
		RefID:   req.GetRefId(),
	})
	if err != nil {
		return nil, mapBalanceBookError(err)
	}

	return &accountv1.DebitBalanceResponse{
		UserId:    balance.UserID,
		Asset:     balance.Asset,
		Available: balance.Available,
		Frozen:    balance.Frozen,
		Total:     balance.Total(),
		Applied:   applied,
	}, nil
}

func mapBalanceBookError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidAsset), errors.Is(err, ErrInvalidAmount):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrInsufficientBalance):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, ErrFreezeNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrFreezeAlreadyClosed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, ErrFreezeAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
