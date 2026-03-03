package domain

type Ipo struct {
	ContractIndex uint32
	AssetName     string
	Address       string
}

type IpoBidData struct {
	ContractIndex uint32
	TickNumber    uint32
	Bids          map[string]int64
}

type TickInfo struct {
	Tick        uint32
	Duration    uint32
	Epoch       uint32
	InitialTick uint32
}

type TickInterval struct {
	First uint32
	Last  uint32
}

type IpoBid struct {
	Price    int64
	Quantity uint16
}

type BidTransaction struct {
	Hash        string
	Amount      uint64
	Source      string
	Destination string
	TickNumber  uint32
	Timestamp   uint64
	InputType   uint32
	InputSize   uint32
	InputData   string // base64
	Signature   string
	MoneyFlew   bool
	Bid         IpoBid
}

type IpoBidTransactions struct {
	AssetName       string
	ContractIndex   uint32
	ContractAddress string
	Transactions    []BidTransaction
}

type IdentityBalance struct {
	Id                         string
	Balance                    int64
	ValidForTick               uint32
	LatestIncomingTransferTick uint32
	LatestOutgoingTransferTick uint32
	IncomingAmount             int64
	OutgoingAmount             int64
	NumberOfIncomingTransfers  uint32
	NumberOfOutgoingTransfers  uint32
}

func GetEpochIntervalsAbsoluteRange(epochIntervals []TickInterval) (first, last uint32) {
	for _, interval := range epochIntervals {
		if interval.First < first || first == 0 {
			first = interval.First
		}
		if interval.Last > last {
			last = interval.Last
		}
	}
	return
}
