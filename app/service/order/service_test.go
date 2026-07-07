package order_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shseooo/go-architecture/app/domain"
	"github.com/shseooo/go-architecture/app/service/order"
)

// --- lightweight stubs (no mock framework needed) ---------------------------

type stubOrderRepo struct {
	created      *domain.Order
	getFn        func(int64) (domain.Order, error)
	statusUpdate func(int64, domain.OrderStatus) error
}

func (s *stubOrderRepo) Create(_ context.Context, o *domain.Order) error {
	o.ID = 1
	s.created = o
	return nil
}
func (s *stubOrderRepo) GetByID(_ context.Context, id int64) (domain.Order, error) {
	return s.getFn(id)
}
func (s *stubOrderRepo) FindByMember(context.Context, int64) ([]domain.Order, error) { return nil, nil }
func (s *stubOrderRepo) UpdateStatus(_ context.Context, id int64, st domain.OrderStatus) error {
	if s.statusUpdate != nil {
		return s.statusUpdate(id, st)
	}
	return nil
}

type stubItemRepo struct {
	items       map[int64]domain.Item
	stockWrites map[int64]int // itemID -> last quantity persisted
}

func (s *stubItemRepo) GetByID(_ context.Context, id int64) (domain.Item, error) {
	it, ok := s.items[id]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	return it, nil
}
func (s *stubItemRepo) UpdateStock(_ context.Context, id int64, q int) error {
	if s.stockWrites == nil {
		s.stockWrites = map[int64]int{}
	}
	s.stockWrites[id] = q
	return nil
}

type stubDeliveryRepo struct{}

func (stubDeliveryRepo) Create(_ context.Context, d *domain.Delivery) error {
	d.ID = 7
	return nil
}

type stubMemberRepo struct{ exists bool }

func (s stubMemberRepo) GetByID(_ context.Context, id int64) (domain.Member, error) {
	if !s.exists {
		return domain.Member{}, domain.ErrNotFound
	}
	return domain.Member{ID: id}, nil
}

// directTx runs fn immediately without a real transaction.
type directTx struct{}

func (directTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func newService(o *stubOrderRepo, i *stubItemRepo, m stubMemberRepo) *order.Service {
	return order.NewService(o, i, stubDeliveryRepo{}, m, directTx{})
}

// --- tests ------------------------------------------------------------------

func TestPlace_Success(t *testing.T) {
	items := &stubItemRepo{items: map[int64]domain.Item{
		1: {ID: 1, Name: "Go Book", Price: 100, StockQuantity: 10, Type: domain.ItemTypeBook},
	}}
	orders := &stubOrderRepo{}
	svc := newService(orders, items, stubMemberRepo{exists: true})

	placed, err := svc.Place(context.Background(), order.PlaceOrderCommand{
		MemberID: 1,
		Address:  domain.Address{City: "Seoul"},
		Lines:    []order.OrderLine{{ItemID: 1, Count: 3}},
	})

	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusOrder, placed.Status)
	assert.Equal(t, 300, placed.TotalPrice())
	assert.Equal(t, 7, int(*placed.DeliveryID))
	assert.Equal(t, domain.DeliveryStatusReady, placed.Delivery.Status)
	// stock decremented 10 -> 7 and persisted
	assert.Equal(t, 7, items.stockWrites[1])
}

func TestPlace_InsufficientStock(t *testing.T) {
	items := &stubItemRepo{items: map[int64]domain.Item{
		1: {ID: 1, Price: 100, StockQuantity: 1, Type: domain.ItemTypeBook},
	}}
	svc := newService(&stubOrderRepo{}, items, stubMemberRepo{exists: true})

	_, err := svc.Place(context.Background(), order.PlaceOrderCommand{
		MemberID: 1,
		Lines:    []order.OrderLine{{ItemID: 1, Count: 5}},
	})

	assert.ErrorIs(t, err, domain.ErrInsufficientStock)
}

func TestPlace_MemberNotFound(t *testing.T) {
	svc := newService(&stubOrderRepo{}, &stubItemRepo{}, stubMemberRepo{exists: false})

	_, err := svc.Place(context.Background(), order.PlaceOrderCommand{
		MemberID: 99,
		Lines:    []order.OrderLine{{ItemID: 1, Count: 1}},
	})

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestPlace_EmptyLines(t *testing.T) {
	svc := newService(&stubOrderRepo{}, &stubItemRepo{}, stubMemberRepo{exists: true})
	_, err := svc.Place(context.Background(), order.PlaceOrderCommand{MemberID: 1})
	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

func TestCancel_RestoresStock(t *testing.T) {
	items := &stubItemRepo{items: map[int64]domain.Item{
		1: {ID: 1, StockQuantity: 5, Type: domain.ItemTypeBook},
	}}
	var canceledTo domain.OrderStatus
	orders := &stubOrderRepo{
		getFn: func(id int64) (domain.Order, error) {
			return domain.Order{
				ID:     id,
				Status: domain.OrderStatusOrder,
				OrderItems: []domain.OrderItem{
					{ItemID: 1, Count: 3, OrderPrice: 100},
				},
			}, nil
		},
		statusUpdate: func(_ int64, st domain.OrderStatus) error { canceledTo = st; return nil },
	}
	svc := newService(orders, items, stubMemberRepo{exists: true})

	err := svc.Cancel(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCancel, canceledTo)
	assert.Equal(t, 8, items.stockWrites[1]) // 5 + 3 restored
}

func TestCancel_AlreadyCanceled(t *testing.T) {
	orders := &stubOrderRepo{
		getFn: func(id int64) (domain.Order, error) {
			return domain.Order{ID: id, Status: domain.OrderStatusCancel}, nil
		},
	}
	svc := newService(orders, &stubItemRepo{}, stubMemberRepo{exists: true})

	err := svc.Cancel(context.Background(), 1)
	assert.ErrorIs(t, err, domain.ErrAlreadyCanceled)
}
