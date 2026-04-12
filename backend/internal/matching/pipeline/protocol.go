package pipeline

import (
	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// SideFlag encodes BUY/SELL as uint8 to avoid string allocs in hot path.
type SideFlag uint8

const (
	SideBuy  SideFlag = 0
	SideSell SideFlag = 1
)

func SideFlagFrom(s model.OrderSide) SideFlag {
	if s == model.OrderSideSell {
		return SideSell
	}
	return SideBuy
}

func (f SideFlag) ToModel() model.OrderSide {
	if f == SideSell {
		return model.OrderSideSell
	}
	return model.OrderSideBuy
}

// TypeFlag encodes LIMIT/MARKET as uint8.
type TypeFlag uint8

const (
	TypeLimit  TypeFlag = 0
	TypeMarket TypeFlag = 1
)

func TypeFlagFrom(t model.OrderType) TypeFlag {
	if t == model.OrderTypeMarket {
		return TypeMarket
	}
	return TypeLimit
}

func (f TypeFlag) ToModel() model.OrderType {
	if f == TypeMarket {
		return model.OrderTypeMarket
	}
	return model.OrderTypeLimit
}

// TIFFlag encodes TimeInForce as uint8.
type TIFFlag uint8

const (
	TIFGTC      TIFFlag = 0
	TIFIOC      TIFFlag = 1
	TIFFOK      TIFFlag = 2
	TIFPostOnly TIFFlag = 3
)

func TIFFlagFrom(t model.TimeInForce) TIFFlag {
	switch t {
	case model.TimeInForceIOC:
		return TIFIOC
	case model.TimeInForceFOK:
		return TIFFOK
	case model.TimeInForcePostOnly:
		return TIFPostOnly
	default:
		return TIFGTC
	}
}

func (f TIFFlag) ToModel() model.TimeInForce {
	switch f {
	case TIFIOC:
		return model.TimeInForceIOC
	case TIFFOK:
		return model.TimeInForceFOK
	case TIFPostOnly:
		return model.TimeInForcePostOnly
	default:
		return model.TimeInForceGTC
	}
}

// STPFlag encodes STPStrategy as uint8.
type STPFlag uint8

const (
	STPFlagNone        STPFlag = 0
	STPFlagCancelTaker STPFlag = 1
	STPFlagCancelMaker STPFlag = 2
	STPFlagCancelBoth  STPFlag = 3
)

func STPFlagFrom(s model.STPStrategy) STPFlag {
	switch s {
	case model.STPCancelTaker:
		return STPFlagCancelTaker
	case model.STPCancelMaker:
		return STPFlagCancelMaker
	case model.STPCancelBoth:
		return STPFlagCancelBoth
	default:
		return STPFlagNone
	}
}

func (f STPFlag) ToModel() model.STPStrategy {
	switch f {
	case STPFlagCancelTaker:
		return model.STPCancelTaker
	case STPFlagCancelMaker:
		return model.STPCancelMaker
	case STPFlagCancelBoth:
		return model.STPCancelBoth
	default:
		return model.STPNone
	}
}

// ActionFlag distinguishes place-order from cancel-orders in the ring buffer.
type ActionFlag uint8

const (
	ActionPlace  ActionFlag = 0
	ActionCancel ActionFlag = 1
)

// MatchCommand travels through the Input Ring Buffer.
// Fields are ordered to minimize struct padding: int64s first, then strings,
// then uint8 flags packed together at the end.
type MatchCommand struct {
	UserID            int64
	MarketID          int64
	Price             int64
	Quantity          int64
	FreezeAmount      int64
	RequestedAtMillis int64

	OrderID         string
	ClientOrderID   string
	Outcome         string
	BookKey         string
	CommandID       string
	TraceID         string
	FreezeID        string
	FreezeAsset     string
	CollateralAsset string

	Action       ActionFlag
	Side         SideFlag
	Type         TypeFlag
	TimeInForce  TIFFlag
	STP          STPFlag
	CancelReason CancelReasonFlag
}

// ToOrder converts the binary-friendly MatchCommand into a model.Order for the engine.
func (c *MatchCommand) ToOrder(nowMillis int64) *model.Order {
	return &model.Order{
		OrderID:         c.OrderID,
		ClientOrderID:   c.ClientOrderID,
		UserID:          c.UserID,
		MarketID:        c.MarketID,
		Outcome:         c.Outcome,
		Side:            c.Side.ToModel(),
		Type:            c.Type.ToModel(),
		TimeInForce:     c.TimeInForce.ToModel(),
		STPStrategy:     c.STP.ToModel(),
		Price:           c.Price,
		Quantity:        c.Quantity,
		CreatedAtMillis: c.RequestedAtMillis,
		UpdatedAtMillis: nowMillis,
	}
}

// ToKafkaCommand reconstructs the original Kafka command for persist/publish.
func (c *MatchCommand) ToKafkaCommand() sharedkafka.OrderCommand {
	return sharedkafka.OrderCommand{
		CommandID:         c.CommandID,
		TraceID:           c.TraceID,
		OrderID:           c.OrderID,
		ClientOrderID:     c.ClientOrderID,
		FreezeID:          c.FreezeID,
		FreezeAsset:       c.FreezeAsset,
		FreezeAmount:      c.FreezeAmount,
		CollateralAsset:   c.CollateralAsset,
		UserID:            c.UserID,
		MarketID:          c.MarketID,
		Outcome:           c.Outcome,
		BookKey:           c.BookKey,
		Side:              string(c.Side.ToModel()),
		Type:              string(c.Type.ToModel()),
		TimeInForce:       string(c.TimeInForce.ToModel()),
		STPStrategy:       string(c.STP.ToModel()),
		Price:             c.Price,
		Quantity:          c.Quantity,
		RequestedAtMillis: c.RequestedAtMillis,
	}
}

// CommandFromKafka converts a decoded Kafka message into a MatchCommand.
// If the upstream OrderService has pre-computed BookKey, it is used directly;
// otherwise we fall back to computing it (backward compatibility).
func CommandFromKafka(cmd sharedkafka.OrderCommand) MatchCommand {
	bookKey := cmd.BookKey
	if bookKey == "" {
		bookKey = model.BuildBookKey(cmd.MarketID, cmd.Outcome)
	}
	return MatchCommand{
		Action:            ActionPlace,
		UserID:            cmd.UserID,
		MarketID:          cmd.MarketID,
		Price:             cmd.Price,
		Quantity:          cmd.Quantity,
		FreezeAmount:      cmd.FreezeAmount,
		RequestedAtMillis: cmd.RequestedAtMillis,
		OrderID:           cmd.OrderID,
		ClientOrderID:     cmd.ClientOrderID,
		Outcome:           cmd.Outcome,
		BookKey:           bookKey,
		CommandID:         cmd.CommandID,
		TraceID:           cmd.TraceID,
		FreezeID:          cmd.FreezeID,
		FreezeAsset:       cmd.FreezeAsset,
		CollateralAsset:   cmd.CollateralAsset,
		Side:              SideFlagFrom(model.OrderSide(cmd.Side)),
		Type:              TypeFlagFrom(model.OrderType(cmd.Type)),
		TimeInForce:       TIFFlagFrom(model.TimeInForce(cmd.TimeInForce)),
		STP:               STPFlagFrom(model.STPStrategy(cmd.STPStrategy)),
	}
}

// CancelReason carried through the ring buffer for cancel actions.
type CancelReasonFlag uint8

const (
	CancelReasonMarketClosed CancelReasonFlag = 1
)

func (f CancelReasonFlag) ToModel() model.CancelReason {
	switch f {
	case CancelReasonMarketClosed:
		return model.CancelReasonMarketClosed
	default:
		return model.CancelReasonNone
	}
}

// MatchResult travels through the Output Ring Buffer.
type MatchResult struct {
	Command  MatchCommand
	Result   engine.Result
	Rejected bool
	EpochID  uint64
}
