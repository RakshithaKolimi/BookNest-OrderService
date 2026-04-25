package postgres

import (
	"context"
	"fmt"

	"booknest-order-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order domain.Order, items []domain.OrderItem) (domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const orderQuery = `
		INSERT INTO orders (
			id,
			order_number,
			user_id,
			total_price,
			payment_method,
			payment_status,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at;
	`

	err = tx.QueryRow(
		ctx,
		orderQuery,
		order.ID,
		order.OrderNumber,
		order.UserID,
		order.TotalPrice,
		order.PaymentMethod,
		order.PaymentStatus,
		order.Status,
	).Scan(&order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return domain.Order{}, fmt.Errorf("insert order: %w", err)
	}

	const itemQuery = `
		INSERT INTO order_items (
			order_id,
			book_id,
			purchase_count,
			purchase_price,
			total_price
		) VALUES ($1, $2, $3, $4, $5);
	`

	for _, item := range items {
		if _, err := tx.Exec(
			ctx,
			itemQuery,
			order.ID,
			item.BookID,
			item.PurchaseCount,
			item.PurchasePrice,
			item.TotalPrice,
		); err != nil {
			return domain.Order{}, fmt.Errorf("insert order item for book %s: %w", item.BookID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit transaction: %w", err)
	}

	return order, nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, orderID string) (domain.Order, []domain.OrderItem, error) {
	const orderQuery = `
		SELECT
			id,
			order_number,
			user_id,
			total_price,
			payment_method,
			payment_status,
			status,
			created_at,
			updated_at
		FROM orders
		WHERE id = $1;
	`

	var order domain.Order
	err := r.db.QueryRow(ctx, orderQuery, orderID).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.UserID,
		&order.TotalPrice,
		&order.PaymentMethod,
		&order.PaymentStatus,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return domain.Order{}, nil, fmt.Errorf("get order %s: %w", orderID, err)
	}

	items, err := r.getOrderItems(ctx, order.ID)
	if err != nil {
		return domain.Order{}, nil, err
	}

	return order, items, nil
}

func (r *OrderRepository) UpdateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	const query = `
		UPDATE orders
		SET
			payment_status = $2,
			status = $3,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			order_number,
			user_id,
			total_price,
			payment_method,
			created_at,
			updated_at;
	`

	err := r.db.QueryRow(
		ctx,
		query,
		order.ID,
		order.PaymentStatus,
		order.Status,
	).Scan(
		&order.OrderNumber,
		&order.UserID,
		&order.TotalPrice,
		&order.PaymentMethod,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return domain.Order{}, fmt.Errorf("update order %s: %w", order.ID, err)
	}

	return order, nil
}

func (r *OrderRepository) ListOrdersByUser(ctx context.Context, userID string, limit, offset int32) ([]domain.OrderWithItems, error) {
	const query = `
		SELECT
			id,
			order_number,
			user_id,
			total_price,
			payment_method,
			payment_status,
			status,
			created_at,
			updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders for user %s: %w", userID, err)
	}
	defer rows.Close()

	orders := make([]domain.OrderWithItems, 0)
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.UserID,
			&order.TotalPrice,
			&order.PaymentMethod,
			&order.PaymentStatus,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order row: %w", err)
		}

		items, err := r.getOrderItems(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		orders = append(orders, domain.OrderWithItems{
			Order: order,
			Items: items,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order rows: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, limit, offset int32) ([]domain.OrderWithItems, error) {
	const query = `
		SELECT
			id,
			order_number,
			user_id,
			total_price,
			payment_method,
			payment_status,
			status,
			created_at,
			updated_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.OrderWithItems, 0)
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.UserID,
			&order.TotalPrice,
			&order.PaymentMethod,
			&order.PaymentStatus,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order row: %w", err)
		}

		items, err := r.getOrderItems(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		orders = append(orders, domain.OrderWithItems{
			Order: order,
			Items: items,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order rows: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) getOrderItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	const query = `
		SELECT
			order_id,
			book_id,
			purchase_count,
			purchase_price,
			total_price
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC;
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items for order %s: %w", orderID, err)
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.OrderID,
			&item.BookID,
			&item.PurchaseCount,
			&item.PurchasePrice,
			&item.TotalPrice,
		); err != nil {
			return nil, fmt.Errorf("scan order item row: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order item rows: %w", err)
	}

	return items, nil
}
