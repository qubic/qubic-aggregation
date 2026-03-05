package integration_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	queryProto "github.com/qubic/archive-query-service/v2/api/archive-query-service/v2"
	statusPb "github.com/qubic/go-data-publisher/status-service/protobuf"
	"github.com/qubic/qubic-aggregation/general-service/clients"
	"github.com/qubic/qubic-aggregation/general-service/domain"
	"github.com/qubic/qubic-http/protobuff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ---------------------------------------------------------------------------
// Fake servers
// ---------------------------------------------------------------------------

type fakeLiveService struct {
	protobuff.UnimplementedQubicLiveServiceServer
	activeIpos []*protobuff.Ipo
	tickInfo   *protobuff.TickInfo
}

func (f *fakeLiveService) GetActiveIpos(_ context.Context, _ *emptypb.Empty) (*protobuff.GetActiveIposResponse, error) {
	return &protobuff.GetActiveIposResponse{Ipos: f.activeIpos}, nil
}

func (f *fakeLiveService) GetTickInfo(_ context.Context, _ *emptypb.Empty) (*protobuff.GetTickInfoResponse, error) {
	return &protobuff.GetTickInfoResponse{TickInfo: f.tickInfo}, nil
}

type fakeStatusService struct {
	statusPb.UnimplementedStatusServiceServer
	intervals []*statusPb.TickInterval
}

func (f *fakeStatusService) GetTickIntervals(_ context.Context, _ *emptypb.Empty) (*statusPb.GetTickIntervalsResponse, error) {
	return &statusPb.GetTickIntervalsResponse{Intervals: f.intervals}, nil
}

type fakeQueryService struct {
	queryProto.UnimplementedArchiveQueryServiceServer

	mu        sync.Mutex
	captured  []*queryProto.GetTransactionsForIdentityRequest
	responses map[string]*queryProto.GetTransactionsForIdentityResponse // keyed by identity
}

func (f *fakeQueryService) GetTransactionsForIdentity(_ context.Context, req *queryProto.GetTransactionsForIdentityRequest) (*queryProto.GetTransactionsForIdentityResponse, error) {
	f.mu.Lock()
	f.captured = append(f.captured, req)
	f.mu.Unlock()

	if resp, ok := f.responses[req.Identity]; ok {
		return resp, nil
	}
	return &queryProto.GetTransactionsForIdentityResponse{
		Hits:         &queryProto.Hits{Total: 0},
		Transactions: nil,
	}, nil
}

// ---------------------------------------------------------------------------
// Test environment
// ---------------------------------------------------------------------------

const bufSize = 1024 * 1024

type testEnv struct {
	fakeLive   *fakeLiveService
	fakeStatus *fakeStatusService
	fakeQuery  *fakeQueryService
	bidService *domain.BidService
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	fakeLive := &fakeLiveService{}
	fakeStatus := &fakeStatusService{}
	fakeQuery := &fakeQueryService{responses: make(map[string]*queryProto.GetTransactionsForIdentityResponse)}

	// Start three in-process gRPC servers via bufconn.
	liveConn := startServer(t, func(s *grpc.Server) {
		protobuff.RegisterQubicLiveServiceServer(s, fakeLive)
	})
	statusConn := startServer(t, func(s *grpc.Server) {
		statusPb.RegisterStatusServiceServer(s, fakeStatus)
	})
	queryConn := startServer(t, func(s *grpc.Server) {
		queryProto.RegisterArchiveQueryServiceServer(s, fakeQuery)
	})

	logger := zap.NewNop().Sugar()
	liveClient := clients.NewLiveServiceClient(liveConn, logger)
	statusClient := clients.NewStatusServiceClient(statusConn, logger)
	queryClient := clients.NewQueryServiceClient(queryConn, logger)

	bidService := domain.NewBidService(logger, liveClient, statusClient, queryClient, 1*time.Minute, 1*time.Minute)

	return &testEnv{
		fakeLive:   fakeLive,
		fakeStatus: fakeStatus,
		fakeQuery:  fakeQuery,
		bidService: bidService,
	}
}

func startServer(t *testing.T, register func(s *grpc.Server)) *grpc.ClientConn {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	register(srv)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.Stop() })

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return conn
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeBidInputData(price int64, quantity uint16) string {
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[0:8], uint64(price))
	binary.LittleEndian.PutUint16(buf[8:10], quantity)
	return base64.StdEncoding.EncodeToString(buf[:])
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestIntegration_CorrectQueryParameters(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Derive the expected SC address for contract index 5.
	expectedAddr, err := domain.ContractIndexToAddress(5)
	require.NoError(t, err)

	// Configure fakes.
	env.fakeLive.activeIpos = []*protobuff.Ipo{
		{ContractIndex: 5, AssetName: "RANDOM"},
	}
	env.fakeLive.tickInfo = &protobuff.TickInfo{
		Tick:        500,
		Duration:    5,
		Epoch:       10,
		InitialTick: 100,
	}
	env.fakeStatus.intervals = []*statusPb.TickInterval{
		{Epoch: 10, FirstTick: 100, LastTick: 300},
		{Epoch: 10, FirstTick: 301, LastTick: 500},
	}

	identity := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	env.fakeQuery.responses[identity] = &queryProto.GetTransactionsForIdentityResponse{
		Hits:         &queryProto.Hits{Total: 1},
		Transactions: []*queryProto.Transaction{{Hash: "h1", InputSize: 16, Amount: 0, InputData: makeBidInputData(42, 1)}},
	}

	_, err = env.bidService.GetCurrentIPOBidTransactions(ctx, []string{identity})
	require.NoError(t, err)

	// Assert captured gRPC request parameters.
	env.fakeQuery.mu.Lock()
	defer env.fakeQuery.mu.Unlock()
	require.Len(t, env.fakeQuery.captured, 1)

	req := env.fakeQuery.captured[0]
	assert.Equal(t, identity, req.Identity)
	assert.Equal(t, expectedAddr, req.Filters["destination"])
	assert.Equal(t, "0", req.Filters["amount"])

	tickRange := req.Ranges["tickNumber"]
	require.NotNil(t, tickRange)

	gte, ok := tickRange.LowerBound.(*queryProto.Range_Gte)
	require.True(t, ok, "expected Range_Gte lower bound")
	assert.Equal(t, "100", gte.Gte)

	lte, ok := tickRange.UpperBound.(*queryProto.Range_Lte)
	require.True(t, ok, "expected Range_Lte upper bound")
	assert.Equal(t, "500", lte.Lte)

	require.NotNil(t, req.Pagination)
	assert.Equal(t, uint32(0), req.Pagination.Offset)
	assert.Equal(t, uint32(1000), req.Pagination.Size)
}

func TestIntegration_EndToEnd(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	addr5, err := domain.ContractIndexToAddress(5)
	require.NoError(t, err)
	addr7, err := domain.ContractIndexToAddress(7)
	require.NoError(t, err)

	env.fakeLive.activeIpos = []*protobuff.Ipo{
		{ContractIndex: 5, AssetName: "ALPHA"},
		{ContractIndex: 7, AssetName: "BETA"},
	}
	env.fakeLive.tickInfo = &protobuff.TickInfo{
		Tick:        600,
		Duration:    5,
		Epoch:       10,
		InitialTick: 100,
	}
	env.fakeStatus.intervals = []*statusPb.TickInterval{
		{Epoch: 10, FirstTick: 100, LastTick: 600},
	}

	id1 := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	id2 := "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

	// id1 has a bid on both IPOs; id2 has a bid on ALPHA only.
	env.fakeQuery.responses[id1] = &queryProto.GetTransactionsForIdentityResponse{
		Hits: &queryProto.Hits{Total: 1},
		Transactions: []*queryProto.Transaction{
			{Hash: "tx1", InputSize: 16, Amount: 0, Source: id1, Destination: addr5, TickNumber: 200, InputData: makeBidInputData(100, 2), MoneyFlew: true},
			{Hash: "tx2", InputSize: 16, Amount: 0, Source: id1, Destination: addr7, TickNumber: 300, InputData: makeBidInputData(200, 3), MoneyFlew: true},
		},
	}
	env.fakeQuery.responses[id2] = &queryProto.GetTransactionsForIdentityResponse{
		Hits: &queryProto.Hits{Total: 1},
		Transactions: []*queryProto.Transaction{
			{Hash: "tx3", InputSize: 16, Amount: 0, Source: id2, Destination: addr5, TickNumber: 400, InputData: makeBidInputData(50, 1), MoneyFlew: false},
		},
	}

	result, err := env.bidService.GetCurrentIPOBidTransactions(ctx, []string{id1, id2})
	require.NoError(t, err)
	require.Len(t, result, 2)

	// ALPHA (contract 5): should have tx1 (from id1) + tx3 (from id2) = both returned since
	// the fake returns the same response regardless of destination filter.
	// Note: in a real scenario the upstream would filter by destination, but our fake
	// returns all configured transactions for the identity. The client does post-filter
	// by InputSize==16 && Amount==0, which all pass here.
	alphaIPO := result[0]
	assert.Equal(t, "ALPHA", alphaIPO.AssetName)
	assert.Equal(t, uint32(5), alphaIPO.ContractIndex)
	assert.Equal(t, addr5, alphaIPO.ContractAddress)

	betaIPO := result[1]
	assert.Equal(t, "BETA", betaIPO.AssetName)
	assert.Equal(t, uint32(7), betaIPO.ContractIndex)
	assert.Equal(t, addr7, betaIPO.ContractAddress)

	// Verify parsed bid data from one of the transactions.
	// ALPHA gets id1's full response (tx1+tx2) and id2's response (tx3).
	// Find a tx with hash "tx1" in ALPHA.
	var foundTx1 *domain.BidTransaction
	for i := range alphaIPO.Transactions {
		if alphaIPO.Transactions[i].Hash == "tx1" {
			foundTx1 = &alphaIPO.Transactions[i]
			break
		}
	}
	require.NotNil(t, foundTx1, "expected tx1 in ALPHA IPO transactions")
	assert.Equal(t, int64(100), foundTx1.Bid.Price)
	assert.Equal(t, uint16(2), foundTx1.Bid.Quantity)
	assert.True(t, foundTx1.MoneyFlew)
}

func TestIntegration_PostFiltering(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	env.fakeLive.activeIpos = []*protobuff.Ipo{
		{ContractIndex: 1, AssetName: "TEST"},
	}
	env.fakeLive.tickInfo = &protobuff.TickInfo{
		Tick:        200,
		Duration:    5,
		Epoch:       1,
		InitialTick: 10,
	}
	env.fakeStatus.intervals = []*statusPb.TickInterval{
		{Epoch: 1, FirstTick: 10, LastTick: 200},
	}

	identity := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	env.fakeQuery.responses[identity] = &queryProto.GetTransactionsForIdentityResponse{
		Hits: &queryProto.Hits{Total: 3},
		Transactions: []*queryProto.Transaction{
			// Valid bid: InputSize==16, Amount==0.
			{Hash: "valid", InputSize: 16, Amount: 0, InputData: makeBidInputData(42, 1)},
			// Wrong InputSize.
			{Hash: "wrong_size", InputSize: 8, Amount: 0, InputData: "AAAAAAAAAAAAAAAA"},
			// Non-zero Amount.
			{Hash: "wrong_amount", InputSize: 16, Amount: 100, InputData: makeBidInputData(99, 2)},
		},
	}

	result, err := env.bidService.GetCurrentIPOBidTransactions(ctx, []string{identity})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Len(t, result[0].Transactions, 1)
	assert.Equal(t, "valid", result[0].Transactions[0].Hash)
	assert.Equal(t, int64(42), result[0].Transactions[0].Bid.Price)
	assert.Equal(t, uint16(1), result[0].Transactions[0].Bid.Quantity)
}
