package service

import (
	"context"
	"testing"
	"time"
	

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/pedidopago/trainingsvc-clients/protos/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	rdb, mock, err := sqlmock.New()
	require.NoError(t, err)

	db := sqlx.NewDb(rdb, "sqlmock")
	service := &Service{
		db: db,
	}
	return service, mock
}

func TestNewClient(t *testing.T) {
	service, mock := newTestService(t)

	mock.ExpectExec("INSERT INTO clients.*").WillReturnResult(sqlmock.NewResult(0, 1))
	resp, err := service.NewClient(context.Background(), &pb.NewClientRequest{
		Name:     "Test",
		Birthday: time.Now().UnixNano(),
		Score:    0,
	})
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAllClients(t *testing.T) {
	service, mock := newTestService(t)

	mock.ExpectExec("DELETE FROM clients.*").WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := service.DeleteAllClients(context.Background(), &pb.DeleteAllClientsRequest{})
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteClient(t *testing.T) {
	service, mock := newTestService(t)

	mock.ExpectExec("DELETE FROM clients.*").WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := service.DeleteClient(context.Background(), &pb.DeleteClientRequest{
		Id: "id",
	})
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewMatch(t *testing.T) {
	service, mock := newTestService(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO client_matches.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE clients.*").WillReturnResult(sqlmock.NewResult(1, 1))
	resp, err := service.NewMatch(context.Background(), &pb.NewMatchRequest{
		ClientId: "idx",
		Score: 	   0,
	})
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryClients(t *testing.T) {
	service, mock := newTestService(t)

	mock.ExpectQuery("SELECT id FROM clients.*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	resp, err := service.QueryClients(context.Background(), &pb.QueryClientsRequest{})
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetClients(t *testing.T) {
	service, mock := newTestService(t)

	query := "SELECT id, name, birthday, score, created_at FROM clients.*"
	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "birthday", "score", "created_at"}))
	resp, err := service.GetClients(context.Background(), &pb.GetClientsRequest{})
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
