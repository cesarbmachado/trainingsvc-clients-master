package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pedidopago/trainingsvc-clients/protos/pb"
	"github.com/pedidopago/trainingsvc-clients/utils"
	"google.golang.org/grpc"
)

type Config struct {
	DBCS string
}

func New(ctx context.Context, sv *grpc.Server, config Config) error {

	svc := &Service{}

	// database connection
	db, err := sqlx.Open("mysql", config.DBCS)
	if err != nil {
		return err
	}
	svc.db = db

	go svc.cleanup(ctx) // executa antes de fechar o app

	pb.RegisterClientsServiceServer(sv, svc)

	return nil
}

type Service struct {
	db *sqlx.DB
}

func (s *Service) cleanup(ctx context.Context) {
	<-ctx.Done()
	s.db.Close()
}

var _ pb.ClientsServiceServer = (*Service)(nil) // compile time check if we support the public proto interface

func (s *Service) NewClient(ctx context.Context, req *pb.NewClientRequest) (*pb.NewClientResponse, error) {
	id := utils.SecureID().String()

	if _, err := s.db.ExecContext(ctx, "INSERT INTO clients (id, name, birthday, score) VALUES (?, ?, ?, ?)",
		id, req.Name, time.Unix(0, req.Birthday), req.Score); err != nil {
		return nil, err
	}

	return &pb.NewClientResponse{
		Id: id,
	}, nil
}

func (s *Service) NewMatch(ctx context.Context, req *pb.NewMatchRequest) (*pb.NewMatchResponse, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Commit()

	if _, err := tx.Exec("INSERT INTO client_matches (client_id, score) VALUES (?, ?)",
		req.ClientId, req.Score); err != nil {
		_ = tx.Rollback()
		return nil, err
	} 

	if _, err := tx.Exec("UPDATE clients SET score = score + ? WHERE id = ?",
		req.Score, req.ClientId); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	return &pb.NewMatchResponse{}, nil
}

func (s *Service) QueryClients(ctx context.Context, req *pb.QueryClientsRequest) (*pb.QueryClientsResponse, error) {
	rq := sq.Select("id").From("clients").OrderBy("score DESC")
	if req.Id != nil {
		rq = rq.Where("id", req.Id.Value)
	}
	if req.Name != nil {
		rq = rq.Where("name LIKE ?", req.Name.Value)
	}
	if req.Birthday != nil {
		rq = req.Birthday.Where("birthday", rq)
	}
	if req.Score != nil {
		rq = req.Score.Where("score", rq)
	}
	if req.CreatedAt != nil {
		rq = req.CreatedAt.Where("created_at", rq)
	}

	query, args, err := rq.ToSql()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	if err := s.db.SelectContext(ctx, &ids, query, args...); err != nil {
		return nil, err
	}

	return &pb.QueryClientsResponse{
		Ids: ids,
	}, nil
}

func (s *Service) GetClients(ctx context.Context, req *pb.GetClientsRequest) (*pb.GetClientsResponse, error) {
	ifids := make([]interface{}, 0, len(req.Ids))
	for _, v := range req.Ids {
		ifids = append(ifids, v)
	}
	q, args, err := sq.Select("id", "name", "birthday", "score", "created_at").From("clients").
		Where(fmt.Sprintf("id IN (%s)", sq.Placeholders(len(ifids))), ifids...).ToSql()
	if err != nil {
		return nil, err
	}
	rawclients := []struct {
		ID        string        `db:"id"`
		Name      string        `db:"name"`
		Birthday  sql.NullTime  `db:"birthday"`
		Score     sql.NullInt64 `db:"score"`
		CreatedAt sql.NullTime  `db:"created_at"`
	}{}
	if err := s.db.SelectContext(ctx, &rawclients, q, args...); err != nil {
		return nil, err
	}
	resp := &pb.GetClientsResponse{
		Clients: make([]*pb.Client, 0, len(rawclients)),
	}
	for _, v := range rawclients {
		resp.Clients = append(resp.Clients, &pb.Client{
			Id:        v.ID,
			Name:      v.Name,
			Birthday:  v.Birthday.Time.UnixNano(),
			Score:     v.Score.Int64,
			CreatedAt: v.CreatedAt.Time.UnixNano(),
		})
	}
	return resp, nil
}

func (s *Service) Sort(ctx context.Context, req *pb.SortRequest) (*pb.SortResponse, error) {
	sorted := req.Items
	sort.Strings(sorted)

	occured := map[string]bool{}
	items := []string{}
	
	if req.RemoveDuplicates == true {
		for str := range sorted {
			if occured[sorted[str]] != true {
				occured[sorted[str]] = true
				items = append(items, sorted[str])
			}
		}
		sorted = items
	}

	return &pb.SortResponse{
		Items: sorted,
	}, nil
}

func (s *Service) DeleteClient(ctx context.Context, req *pb.DeleteClientRequest) (*pb.DeleteClientResponse, error) {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM clients WHERE id = ?", req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteClientResponse{}, nil
}

func (s *Service) DeleteAllClients(ctx context.Context, req *pb.DeleteAllClientsRequest) (*pb.DeleteAllClientsResponse, error) {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM clients"); err != nil {
		return nil, err
	}
	return &pb.DeleteAllClientsResponse{}, nil
}
